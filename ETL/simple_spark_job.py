from pyspark.sql import SparkSession


def main():
    spark = SparkSession.builder \
        .appName("ReadFromMinIO") \
        .master("spark://spark-master:7077") \
    .getOrCreate()

    s3_path = "s3a://test-bucket/data.json"

    try:
        df = spark.read.json(s3_path)

        print("=== Schema ===")
        df.printSchema()

        print("=== Data ===")
        df.show()

        count = df.count()
        print(f"=== Total rows: {count} ===")

    except Exception as e:
        print(f"Error: {str(e)}")
        print("\nTroubleshooting steps:")
        print("1. Make sure MinIO is running and accessible")
        print("2. Create 'test-bucket' in MinIO")
        print("3. Upload a JSON file named 'data.json' to the bucket")
        print("4. Check MinIO credentials in spark-defaults.conf")

    spark.stop()


if __name__ == "__main__":
    main()