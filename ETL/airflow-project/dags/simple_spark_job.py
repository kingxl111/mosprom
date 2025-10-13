from pyspark.sql import SparkSession
from pyspark.sql.functions import col

def main():
    spark = SparkSession.builder \
        .appName("MinIO_to_ClickHouse") \
        .config("spark.sql.catalog.clickhouse_catalog", "com.clickhouse.spark.ClickHouseCatalog") \
        .config("spark.sql.catalog.clickhouse_catalog.host", "clickhouse") \
        .config("spark.sql.catalog.clickhouse_catalog.http_port", "8123") \
        .config("spark.sql.catalog.clickhouse_catalog.user", "admin") \
        .config("spark.sql.catalog.clickhouse_catalog.password", "clickhouse123") \
        .config("spark.sql.catalog.clickhouse_catalog.database", "default") \
        .getOrCreate()

    s3_path = "s3a://test-bucket/data.json"

    try:
        df_raw = spark.read.json(s3_path)

        print("=== Исходные данные (с ошибками) ===")
        df_raw.show(truncate=False)

        df_clean = df_raw.filter(col("_corrupt_record").isNull()).select("technology", "score")

        print("=== Очищенные данные ===")
        df_clean.show()

        df_clean.createOrReplaceTempView("temp_technologies")
        spark.sql("""
            INSERT INTO clickhouse_catalog.default.technologies
            SELECT * FROM temp_technologies
        """)

        print("✅ Данные успешно записаны в ClickHouse!")

        result = spark.table("clickhouse_catalog.default.technologies")
        print("=== Данные из ClickHouse ===")
        result.show()

    except Exception as e:
        print(f"Ошибка: {str(e)}")
        import traceback
        traceback.print_exc()

    finally:
        spark.stop()


if __name__ == "__main__":
    main()