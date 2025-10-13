from kafka import KafkaConsumer
import json
import time
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class OptimizedHighThroughputConsumer:
    def __init__(self):
        self.consumer = KafkaConsumer(
            'high-throughput-topic',
            bootstrap_servers='localhost:9092',

            fetch_max_bytes=104857600,
            fetch_min_bytes=131072,
            fetch_max_wait_ms=100,
            max_partition_fetch_bytes=2097152,

            max_poll_records=5000,

            auto_offset_reset='earliest',
            value_deserializer=lambda m: json.loads(m.decode('utf-8')),
            key_deserializer=lambda m: m.decode('utf-8') if m else None
        )

        self.processed_count = 0
        self.start_time = None
        self.running = False
        self.total_batches = 0
        self.total_batch_time = 0

    def process_message_batch(self, messages):
        batch_size = len(messages)

        for message in messages:
            if 'id' in message.value:
                self.processed_count += 1
            else:
                logger.warning(f"Missing ID in message offset {message.offset}")

        return batch_size

    def start_consuming(self):
        print("Starting OPTIMIZED high-throughput consumer...")
        print(f"Initial partitions: {self.consumer.assignment()}")

        self.running = True
        self.start_time = time.time()
        last_stats_time = self.start_time
        last_processed = 0

        try:
            while self.running:
                batch_start = time.time()

                records = self.consumer.poll(timeout_ms=500)

                if records:
                    batch_size = 0
                    for topic_partition, messages in records.items():
                        processed_in_batch = self.process_message_batch(messages)
                        batch_size += processed_in_batch

                    batch_time = time.time() - batch_start
                    self.total_batches += 1
                    self.total_batch_time += batch_time

                    if batch_size > 1000 or batch_time > 0.01:
                        speed = batch_size / batch_time if batch_time > 0 else 0
                        print(f" Batch: {batch_size} msg in {batch_time:.4f}s ({speed:,.0f} msg/sec)")

                current_time = time.time()
                if self.processed_count - last_processed >= 100000:
                    elapsed = current_time - self.start_time
                    overall_rate = self.processed_count / elapsed

                    recent_elapsed = current_time - last_stats_time
                    recent_rate = (self.processed_count - last_processed) / recent_elapsed

                    print(f"Progress: {self.processed_count:,} messages | "
                          f"Overall: {overall_rate:,.0f} msg/sec | "
                          f"Recent: {recent_rate:,.0f} msg/sec")

                    last_stats_time = current_time
                    last_processed = self.processed_count

                if self.processed_count >= 100000000000:
                    print("Reached target of 10 million messages")
                    break


                if current_time - self.start_time > 300:
                    print("Time limit reached (5 minutes)")
                    break

        except KeyboardInterrupt:
            print("\nStopping consumer...")
        except Exception as e:
            logger.error(f" Consumer error: {e}")



def main():
    consumer = OptimizedHighThroughputConsumer()
    consumer.start_consuming()


if __name__ == "__main__":
    main()
