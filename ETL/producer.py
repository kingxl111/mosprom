from kafka import KafkaProducer
import json
import time

producer = KafkaProducer(
    bootstrap_servers='localhost:9092',
    batch_size=163840,
    linger_ms=20,

    compression_type='lz4',

    buffer_memory=335544320,

    acks='all',
    retries=10,
    retry_backoff_ms=1000,
    request_timeout_ms=30000,

    value_serializer=lambda v: json.dumps(v).encode('utf-8'),
    key_serializer=lambda v: str(v).encode('utf-8') if v else None
)


def main():
    total_messages = 1000000

    print(f"Starting high-throughput producer...")
    print(f"Target: {total_messages} messages")
    start_time = time.time()

    try:
        for i in range(total_messages):
            message = {
                "id": i,
                "timestamp": time.time(),
                "data": "x" * 100
            }

            key = i % 100

            future = producer.send('high-throughput-topic', key=key, value=message)

            if i % 10000 == 0:
                record_metadata = future.get(timeout=10)
                elapsed = time.time() - start_time
                rate = i / elapsed if elapsed > 0 else 0
                print(f"Sent {i}/{total_messages} messages "
                      f"({rate:.0f} msg/sec, partition {record_metadata.partition})")

    except Exception as e:
        print(f"Error: {e}")

    finally:
        producer.flush()
        producer.close()

        end_time = time.time()
        total_time = end_time - start_time
        print(f"Producer completed: {total_messages} messages in {total_time:.2f}s "
              f"({total_messages / total_time:.0f} msg/sec)")


if __name__ == "__main__":
    main()
