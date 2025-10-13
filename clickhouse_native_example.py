from pyspark.sql import SparkSession
from pyspark.sql.types import StructType, StructField, LongType, StringType, TimestampType

spark = SparkSession.builder \
    .appName("ClickHouse Native Example") \
    .master("spark://spark-master:7077") \
    .config("spark.sql.catalog.clickhouse_catalog", "com.clickhouse.spark.ClickHouseCatalog") \
    .config("spark.sql.catalog.clickhouse_catalog.host", "clickhouse") \
    .config("spark.sql.catalog.clickhouse_catalog.http_port", 8123) \
    .config("spark.sql.catalog.clickhouse_catalog.user", "admin") \
    .config("spark.sql.catalog.clickhouse_catalog.password", "clickhouse123") \
    .config("spark.sql.catalog.clickhouse_catalog.database", "default") \
    .getOrCreate()

print("=== Проверка подключения ===")
df_test = spark.sql("SELECT * FROM clickhouse_catalog.system.numbers LIMIT 5")
df_test.show()

print("=== Создание таблицы ===")
spark.sql("""
    CREATE TABLE IF NOT EXISTS clickhouse_catalog.default.my_first_spark_table (
        id Long NOT NULL,
        name String,
        event_time Timestamp
    ) USING ClickHouse
    TBLPROPERTIES (
        engine = 'MergeTree()',
        order_by = 'id'
    )
""")

print("=== Запись данных в таблицу ===")
data = [
    (1, "Алексей", "2024-01-01 10:00:00"),
    (2, "Мария", "2024-01-01 10:01:00"),
    (3, "Иван", "2024-01-01 10:02:00")
]

temp_df = spark.createDataFrame(data, ["id", "name", "event_time"])
temp_df.createOrReplaceTempView("temp_data")

spark.sql("""
    INSERT INTO clickhouse_catalog.default.my_first_spark_table
    SELECT * FROM temp_data
""")

print("=== Чтение данных из таблицы ===")
df_readback = spark.table("clickhouse_catalog.default.my_first_spark_table")
df_readback.show()

print("Успех! Нативный коннектор работает!")
spark.stop()