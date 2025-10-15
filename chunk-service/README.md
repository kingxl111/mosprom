1. docker-compose -f docker-compose.simple.yml up -d kafka-broker1 kafka-broker2 kafka-broker3 kafka-ui minio kafka-connect 

2. docker exec -it kafka-broker1 /opt/kafka/bin/kafka-topics.sh --create \
    --topic file-chunks-topic \
    --partitions 3 \
    --replication-factor 3 \
    --bootstrap-server localhost:9092

3. Создайте бакет:
Это в minio
Нажмите "+" (Create Bucket)

Имя бакета: moscow-industry-data

Сохраните

4. curl -X POST http://localhost:8083/connectors -H "Content-Type: application/json" -d @chan-updated-connector.json