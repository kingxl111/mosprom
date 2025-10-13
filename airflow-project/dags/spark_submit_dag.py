
from airflow import DAG
from airflow.providers.apache.spark.operators.spark_submit import SparkSubmitOperator
from datetime import datetime, timedelta

default_args = {
    'owner': 'airflow',
    'start_date': datetime(2024, 1, 1),
    'retries': 1,
    'retry_delay': timedelta(minutes=5),
}

with DAG(
    'spark_submit_dag',
    default_args=default_args,
    description='Submit simple_spark_job.py to Spark cluster',
    schedule_interval=timedelta(minutes=30),
    catchup=False,
    tags=['spark'],
) as dag:

    submit_spark_job = SparkSubmitOperator(
        task_id='submit_simple_spark_job',
        application='/opt/airflow/dags/simple_spark_job.py',
        conn_id='spark_default',
        name='simple-spark-job',
        verbose=True,
        jars="/opt/spark/jars/hadoop-aws-3.3.4.jar,/opt/spark/jars/aws-java-sdk-bundle-1.12.262.jar,/opt/spark/jars/clickhouse-jdbc-0.6.3-all.jar,/opt/spark/jars/clickhouse-spark-runtime-3.5_2.12-0.8.1.jar",
        conf={
            "spark.hadoop.fs.s3a.impl": "org.apache.hadoop.fs.s3a.S3AFileSystem",
            "spark.hadoop.fs.s3a.endpoint": "http://minio:9000",
            "spark.hadoop.fs.s3a.access.key": "minioadmin",
            "spark.hadoop.fs.s3a.secret.key": "minioadmin123",
            "spark.hadoop.fs.s3a.path.style.access": "true",
            "spark.hadoop.fs.s3a.connection.ssl.enabled": "false",
            "spark.hadoop.fs.s3a.aws.credentials.provider": "org.apache.hadoop.fs.s3a.SimpleAWSCredentialsProvider",
            "spark.jars": "/opt/spark/jars/hadoop-aws-3.3.4.jar,/opt/spark/jars/aws-java-sdk-bundle-1.12.262.jar,/opt/spark/jars/clickhouse-jdbc-0.6.3-all.jar,/opt/spark/jars/clickhouse-spark-runtime-3.5_2.12-0.8.1.jar",
            "spark.driver.extraClassPath": "/opt/spark/jars/hadoop-aws-3.3.4.jar:/opt/spark/jars/aws-java-sdk-bundle-1.12.262.jar:/opt/spark/jars/clickhouse-jdbc-0.6.3-all.jar:/opt/spark/jars/clickhouse-spark-runtime-3.5_2.12-0.8.1.jar",
            "spark.executor.extraClassPath": "/opt/spark/jars/hadoop-aws-3.3.4.jar:/opt/spark/jars/aws-java-sdk-bundle-1.12.262.jar:/opt/spark/jars/clickhouse-jdbc-0.6.3-all.jar:/opt/spark/jars/clickhouse-spark-runtime-3.5_2.12-0.8.1.jar",
            "spark.sql.catalog.clickhouse_catalog": "com.clickhouse.spark.ClickHouseCatalog",
            "spark.sql.catalog.clickhouse_catalog.host": "clickhouse",
            "spark.sql.catalog.clickhouse_catalog.http_port": "8123",
            "spark.sql.catalog.clickhouse_catalog.user": "admin",
            "spark.sql.catalog.clickhouse_catalog.password": "clickhouse123",
            "spark.sql.catalog.clickhouse_catalog.database": "default"
        }
    )

    submit_spark_job

"""
from airflow import DAG
from airflow.providers.apache.spark.operators.spark_submit import SparkSubmitOperator
from datetime import datetime, timedelta

default_args = {
    'owner': 'airflow',
    'start_date': datetime(2024, 1, 1),
    'retries': 1,
    'retry_delay': timedelta(minutes=5),
}

with DAG(
    'spark_submit_dag',
    default_args=default_args,
    description='Submit simple_spark_job.py to Spark cluster',
    schedule_interval=timedelta(minutes=30),
    catchup=False,
    tags=['spark'],
) as dag:

    submit_spark_job = SparkSubmitOperator(
        task_id='submit_simple_spark_job',
        application='/opt/airflow/dags/simple_spark_job.py',
        conn_id='spark_default',
        name='simple-spark-job',
        verbose=True,
        jars="/opt/spark/jars/hadoop-aws-3.3.4.jar,/opt/spark/jars/aws-java-sdk-bundle-1.12.262.jar",
        conf={
            "spark.hadoop.fs.s3a.impl": "org.apache.hadoop.fs.s3a.S3AFileSystem",
            "spark.hadoop.fs.s3a.endpoint": "http://minio:9000",
            "spark.hadoop.fs.s3a.access.key": "minioadmin",
            "spark.hadoop.fs.s3a.secret.key": "minioadmin123",
            "spark.hadoop.fs.s3a.path.style.access": "true",
            "spark.hadoop.fs.s3a.connection.ssl.enabled": "false",
            "spark.hadoop.fs.s3a.aws.credentials.provider": "org.apache.hadoop.fs.s3a.SimpleAWSCredentialsProvider",
            "spark.jars": "/opt/spark/jars/hadoop-aws-3.3.4.jar,/opt/spark/jars/aws-java-sdk-bundle-1.12.262.jar",
            "spark.driver.extraClassPath": "/opt/spark/jars/hadoop-aws-3.3.4.jar:/opt/spark/jars/aws-java-sdk-bundle-1.12.262.jar",
            "spark.executor.extraClassPath": "/opt/spark/jars/hadoop-aws-3.3.4.jar:/opt/spark/jars/aws-java-sdk-bundle-1.12.262.jar"
        }
    )

    submit_spark_job
"""
