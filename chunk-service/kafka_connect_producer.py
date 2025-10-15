import json
import os
from datetime import datetime
from kafka import KafkaProducer
from kafka.errors import KafkaError, KafkaTimeoutError
import hashlib
import time

class KafkaConnectFileProducer:
    def __init__(self, bootstrap_servers='localhost:9092'):
        self.producer = KafkaProducer(
            bootstrap_servers=bootstrap_servers,
            batch_size=163840,  # 160KB
            linger_ms=50,
            compression_type='lz4',
            value_serializer=lambda v: json.dumps(v, default=str).encode('utf-8'),
            key_serializer=lambda v: str(v).encode('utf-8') if v else None,
            acks='all',
            request_timeout_ms=30000,
            retries=5,
            retry_backoff_ms=500,
            max_block_ms=60000,
            buffer_memory=33554432
        )

        self.global_send_counter = 0

    def get_next_send_id(self):
        """Возвращает следующий глобальный номер отправки"""
        self.global_send_counter += 1
        return self.global_send_counter

    def create_message_with_schema(self, payload):
        """Создает сообщение с JSON-схемой для Kafka Connect"""
        schema = {
            "type": "struct",
            "fields": [
                {"type": "string", "optional": False, "field": "enterprise_id"},
                {"type": "string", "optional": False, "field": "file_name"},
                {"type": "string", "optional": False, "field": "file_extension"},
                {"type": "int32", "optional": False, "field": "chunk_id"},
                {"type": "int32", "optional": False, "field": "total_chunks"},
                {"type": "string", "optional": False, "field": "chunk_data"},
                {"type": "string", "optional": False, "field": "file_hash"},
                {"type": "int32", "optional": False, "field": "file_size"},
                {"type": "string", "optional": False, "field": "processing_date"},
                {"type": "string", "optional": False, "field": "timestamp"},
                {"type": "int32", "optional": False, "field": "chunk_size"},
                {"type": "string", "optional": False, "field": "path_s3"}
            ],
            "optional": False,
            "name": "file_chunk"
        }

        return {
            "schema": schema,
            "payload": payload
        }

    def send_message_with_confirmation(self, topic, key, value, timeout=30):
        """Отправляет сообщение с подтверждением доставки и повторными попытками"""
        max_retries = 3
        attempt = 0
        while attempt < max_retries:
            try:
                future = self.producer.send(topic, key=key, value=value)
                record_metadata = future.get(timeout=timeout)
                print(
                    f"Message delivered to {record_metadata.topic} [{record_metadata.partition}] at offset {record_metadata.offset}")
                return True
            except KafkaTimeoutError as e:
                attempt += 1
                print(f"Attempt {attempt} failed to send message due to timeout: {e}")
                if attempt < max_retries:
                    print(f"Retrying in {attempt} seconds...")
                    time.sleep(attempt)  # Exponential backoff
                else:
                    print(f"Failed to send message after {max_retries} attempts: {e}")
                    return False
            except KafkaError as e:
                print(f"Kafka error occurred: {e}")
                return False
            except Exception as e:
                print(f"Unexpected error occurred: {e}")
                return False
        return False

    def process_file(self, file_path, enterprise_id, chunk_size=1024 * 1024):
        """Обрабатывает файл и отправляет все чанки с одинаковым временем"""
        file_name = os.path.basename(file_path)
        file_extension = file_name.split('.')[-1] if '.' in file_name else 'bin'

        with open(file_path, 'rb') as f:
            file_data = f.read()

        file_hash = hashlib.md5(file_data).hexdigest()
        total_chunks = (len(file_data) + chunk_size - 1) // chunk_size

        send_id = self.get_next_send_id()

        current_datetime = datetime.now()
        current_date = current_datetime.strftime("%Y-%m-%d")

        print(f"Processing {file_name} for enterprise {enterprise_id}, total chunks: {total_chunks}")
        print(f"Global Send ID: {send_id}")

        successful_chunks = 0
        chunk_futures = []
        for chunk_id in range(total_chunks):
            start = chunk_id * chunk_size
            end = min(start + chunk_size, len(file_data))
            chunk_data = file_data[start:end]

            path_s3 = f"/data/{current_date}/enterprise_{enterprise_id}_send_{send_id}/chunk_{chunk_id + 1}"

            payload = {
                "enterprise_id": enterprise_id,
                "file_name": file_name,
                "file_extension": file_extension,
                "chunk_id": chunk_id,
                "total_chunks": total_chunks,
                "chunk_data": chunk_data.hex(),
                "file_hash": file_hash,
                "file_size": len(file_data),
                "processing_date": current_date,
                "timestamp": current_datetime.isoformat(),
                "chunk_size": len(chunk_data),
                "path_s3": path_s3
            }

            message = self.create_message_with_schema(payload)

            key = f"{enterprise_id}_{current_date}_{send_id}"

            if self.send_message_with_confirmation('file-chunks-topic', key, message):
                successful_chunks += 1


            if (chunk_id + 1) % 10 == 0:
                print(f"Sent chunk {chunk_id + 1}/{total_chunks}")

        if successful_chunks == total_chunks:

            print(f"All {total_chunks} chunks sent successfully. Now sending load message.")

            if self.send_load_message(enterprise_id, send_id, current_datetime):
                print(f"File {file_name} successfully sent in {total_chunks} chunks (Global Send ID: {send_id})")
                return True
            else:
                print(f"Failed to send load message for {file_name}")
                return False
        else:
            print(f"Failed to send all chunks for {file_name}. Sent {successful_chunks}/{total_chunks}")
            return False

    def send_load_message(self, enterprise_id, send_id, processing_datetime):
        """Отправляет сообщение о завершении загрузки в папку status"""
        current_date = processing_datetime.strftime("%Y-%m-%d")

        path_s3 = f"/status/enterprise_{enterprise_id}_send_{send_id}_load"

        payload = {
            "type": "load",
            "enterprise_id": enterprise_id,
            "send_id": send_id,
            "processing_date": current_date,
            "timestamp": processing_datetime.isoformat(),
            "path_s3": path_s3
        }

        schema = {
            "type": "struct",
            "fields": [
                {"type": "string", "optional": False, "field": "type"},
                {"type": "string", "optional": False, "field": "enterprise_id"},
                {"type": "int32", "optional": False, "field": "send_id"},
                {"type": "string", "optional": False, "field": "processing_date"},
                {"type": "string", "optional": False, "field": "timestamp"},
                {"type": "string", "optional": False, "field": "path_s3"}
            ],
            "optional": False,
            "name": "file_load"
        }

        message = {
            "schema": schema,
            "payload": payload
        }

        key = f"{enterprise_id}_{current_date}_{send_id}_load"

        if self.send_message_with_confirmation('file-chunks-topic', key, message, timeout=45):
            print(f"Successfully sent load message for enterprise {enterprise_id} (Global Send ID: {send_id})")
            return True
        else:
            print(f"Failed to send load message for enterprise {enterprise_id} (Global Send ID: {send_id})")
            return False

    def close(self):
        """Closes the Kafka producer"""
        if self.producer:
            print("Flushing producer...")
            self.producer.flush(timeout=30)
            print("Producer flushed successfully.")
            self.producer.close()
            print("Producer closed.")


def main():
    try:
        producer = KafkaConnectFileProducer()
        print("Successfully connected to Kafka")

        test_data_dir = "test_data"
        os.makedirs(f"{test_data_dir}/enterprise_12345", exist_ok=True)
        os.makedirs(f"{test_data_dir}/enterprise_99999", exist_ok=True)

        csv_content = """timestamp,sensor_id,value,unit
2024-01-15 10:00:00,sensor_001,25.5,temperature
2024-01-15 10:01:00,sensor_002,65.8,humidity
2024-01-15 10:02:00,sensor_001,25.7,temperature
2024-01-15 10:03:00,sensor_003,1013.2,pressure"""

        with open(f"{test_data_dir}/enterprise_12345/data.csv", "w") as f:
            f.write(csv_content)

        json_content = {
            "enterprise_id": "99999",
            "sensors": [
                {"id": "sensor_001", "type": "temperature", "location": "building_a"},
                {"id": "sensor_002", "type": "humidity", "location": "building_b"}
            ],
            "update_frequency": "1m"
        }

        with open(f"{test_data_dir}/enterprise_99999/config.json", "w") as f:
            json.dump(json_content, f, indent=2)

        files_to_process = [
            {'path': f'{test_data_dir}/enterprise_12345/data.csv', 'enterprise_id': '12345'},
            {'path': f'{test_data_dir}/enterprise_99999/config.json', 'enterprise_id': '99999'},
        ]

        successful_sends = 0
        total_sends = 0

        for file_info in files_to_process:
            if os.path.exists(file_info['path']):
                for i in range(5):
                    total_sends += 1
                    print(f"\n--- Sending file {file_info['path']}, attempt {i + 1}/5 ---")
                    try:
                        if producer.process_file(file_info['path'], file_info['enterprise_id'],
                                                 chunk_size=50):
                            successful_sends += 1
                        time.sleep(1)
                    except Exception as e:
                        print(f"Failed to process file {file_info['path']} on attempt {i + 1}: {e}")
            else:
                print(f"File not found: {file_info['path']}")

        print(f"\n=== Summary ===")
        print(f"Successful sends: {successful_sends}/{total_sends}")

    except Exception as e:
        print(f"Critical error in main: {e}")
    finally:
        if 'producer' in locals() and producer:
            producer.close()


if __name__ == "__main__":
    main()