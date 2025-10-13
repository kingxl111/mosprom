from airflow import DAG
from airflow.operators.bash import BashOperator
from airflow.operators.python import PythonOperator
from datetime import datetime, timedelta

default_args = {
    'owner': 'airflow',
    'depends_on_past': False,
    'start_date': datetime(2024, 1, 1),
    'retries': 1,
    'retry_delay': timedelta(minutes=5),
}

with DAG(
    'my_first_dag',
    default_args=default_args,
    description='A simple tutorial DAG',
    schedule_interval=timedelta(seconds=30),
    catchup=False,
    tags=['example'],
) as dag:

    t1 = BashOperator(
        task_id='print_date',
        bash_command='date',
    )

    t2 = BashOperator(
        task_id='sleep',
        bash_command='sleep 5',
        retries=3,
    )

    def print_context():
        print("Добро пожаловать в Apache Airflow!")

    t3 = PythonOperator(
        task_id='print_welcome',
        python_callable=print_context,
    )


    t1 >> [t2, t3]