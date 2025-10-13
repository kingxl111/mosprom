FROM apache/spark:3.5.1

USER root

RUN apt-get update && apt-get install -y curl && apt-get clean

RUN curl -L https://repo1.maven.org/maven2/org/apache/hadoop/hadoop-aws/3.3.4/hadoop-aws-3.3.4.jar -o /opt/spark/jars/hadoop-aws-3.3.4.jar && \
    curl -L https://repo1.maven.org/maven2/com/amazonaws/aws-java-sdk-bundle/1.12.262/aws-java-sdk-bundle-1.12.262.jar -o /opt/spark/jars/aws-java-sdk-bundle-1.12.262.jar

RUN curl -L https://repo1.maven.org/maven2/com/clickhouse/spark/clickhouse-spark-runtime-3.5_2.12/0.8.1/clickhouse-spark-runtime-3.5_2.12-0.8.1.jar -o /opt/spark/jars/clickhouse-spark-runtime-3.5_2.12-0.8.1.jar
RUN curl -L https://repo1.maven.org/maven2/com/clickhouse/clickhouse-jdbc/0.6.3/clickhouse-jdbc-0.6.3-all.jar -o /opt/spark/jars/clickhouse-jdbc-0.6.3-all.jar

COPY spark-defaults.conf /opt/spark/conf/
#COPY jars/*.jar /opt/spark/jars/

USER ${spark_uid}