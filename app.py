# app.py
import streamlit as st
import requests
import json
import time
from datetime import datetime
import io
import pandas as pd
import plotly.express as px

# Конфигурация
st.set_page_config(
    page_title="Moscow Industrial Platform",
    page_icon="🏭",
    layout="wide",
    initial_sidebar_state="expanded"
)

# Глобальные настройки
DEMO_MODE = True  # Переключите на False когда бэкенд будет готов
API_BASE_URL = "http://localhost:8080"

# Состояние приложения
if 'access_token' not in st.session_state:
    st.session_state.access_token = None
if 'user_info' not in st.session_state:
    st.session_state.user_info = None
if 'demo_files' not in st.session_state:
    st.session_state.demo_files = []
if 'demo_reports' not in st.session_state:
    st.session_state.demo_reports = []
if 'demo_mode' not in st.session_state:
    st.session_state.demo_mode = DEMO_MODE

# Демо-данные
def create_demo_user():
    return {
        "id": "demo-user-123",
        "email": "demo@industrial.ru",
        "role": "company",
        "company_name": "Демо Промышленный Завод",
        "contact_person": "Иван Демов",
        "phone": "+7 495 123-45-67",
        "created_at": "2024-01-15T10:30:00Z"
    }

def demo_auth_response():
    return {
        "access_token": "demo-jwt-token-12345",
        "refresh_token": "demo-refresh-token-12345",
        "expires_in": 3600,
        "token_type": "Bearer",
        "user": create_demo_user()
    }

def make_authenticated_request(method, endpoint, **kwargs):
    """Вспомогательная функция для авторизованных запросов"""
    
    # Демо-режим
    if st.session_state.demo_mode:
        return handle_demo_request(method, endpoint, **kwargs)
    
    # Реальный режим
    headers = kwargs.get('headers', {})
    if st.session_state.access_token:
        headers['Authorization'] = f'Bearer {st.session_state.access_token}'
    kwargs['headers'] = headers
    
    url = f"{API_BASE_URL}{endpoint}"
    
    try:
        if method == 'GET':
            response = requests.get(url, **kwargs)
        elif method == 'POST':
            response = requests.post(url, **kwargs)
        elif method == 'PUT':
            response = requests.put(url, **kwargs)
        elif method == 'DELETE':
            response = requests.delete(url, **kwargs)
        else:
            return None
        
        return response
    except requests.exceptions.RequestException as e:
        st.error(f"Ошибка соединения: {e}")
        st.info("💡 Совет: Запустите бэкенд сервер или используйте демо-режим")
        return None

def handle_demo_request(method, endpoint, **kwargs):
    """Обработка запросов в демо-режиме"""
    
    class DemoResponse:
        def __init__(self, status_code, data):
            self.status_code = status_code
            self._data = data
            
        def json(self):
            return self._data
    
    # Аутентификация
    if endpoint == '/api/v1/auth/register' and method == 'POST':
        st.session_state.access_token = "demo-jwt-token"
        st.session_state.user_info = create_demo_user()
        return DemoResponse(201, demo_auth_response())
        
    elif endpoint == '/api/v1/auth/login' and method == 'POST':
        st.session_state.access_token = "demo-jwt-token"
        st.session_state.user_info = create_demo_user()
        return DemoResponse(200, demo_auth_response())
    
    elif endpoint == '/api/v1/auth/logout' and method == 'POST':
        st.session_state.access_token = None
        st.session_state.user_info = None
        return DemoResponse(204, {})
    
    # Пользователь
    elif endpoint == '/api/v1/users/me' and method == 'GET':
        return DemoResponse(200, create_demo_user())
    
    # Файлы
    elif endpoint == '/api/v1/files/upload' and method == 'POST':
        file_id = f"file-{len(st.session_state.demo_files) + 1}"
        files = kwargs.get('files', {})
        filename = files[0][0] if files else "demo.txt"
        
        new_file = {
            "file_id": file_id,
            "filename": filename,
            "status": "processing",
            "message": "Файл принят в обработку",
            "created_at": datetime.now().isoformat(),
            "estimated_processing_time": 30
        }
        
        st.session_state.demo_files.append(new_file)
        return DemoResponse(202, new_file)
    
    elif endpoint == '/api/v1/files' and method == 'GET':
        # Обновляем статусы файлов для демо
        for file in st.session_state.demo_files:
            if file['status'] == 'processing':
                # Через 5 секунд после загрузки меняем статус на processed
                upload_time = datetime.fromisoformat(file['created_at'])
                if (datetime.now() - upload_time).seconds > 5:
                    file['status'] = 'processed'
                    file['message'] = 'Файл успешно обработан'
        
        files_data = {
            "files": st.session_state.demo_files,
            "pagination": {
                "page": 1,
                "limit": 20,
                "total": len(st.session_state.demo_files),
                "total_pages": 1
            }
        }
        return DemoResponse(200, files_data)
    
    # Аналитика
    elif endpoint == '/api/v1/analytics/reports' and method == 'GET':
        reports_data = {
            "reports": st.session_state.demo_reports,
            "pagination": {
                "page": 1,
                "limit": 10,
                "total": len(st.session_state.demo_reports),
                "total_pages": 1
            }
        }
        return DemoResponse(200, reports_data)
    
    elif endpoint == '/api/v1/analytics/reports/generate' and method == 'POST':
        report_id = f"report-{len(st.session_state.demo_reports) + 1}"
        report_data = kwargs.get('json', {})
        
        new_report = {
            "report_id": report_id,
            "status": "queued",
            "estimated_completion_time": 60,
            "message": "Отчет поставлен в очередь на генерацию"
        }
        
        # Добавляем в историю после "генерации"
        completed_report = {
            "report_id": report_id,
            "report_type": report_data.get('report_type', 'comprehensive'),
            "report_name": f"Отчет {report_data.get('report_type', 'comprehensive')}",
            "generated_at": datetime.now().isoformat(),
            "time_range": report_data.get('time_range', {}),
            "file_size": 1024 * 5,  # 5KB
            "download_url": "#"
        }
        
        st.session_state.demo_reports.append(completed_report)
        return DemoResponse(202, new_report)
    
    # Система
    elif endpoint == '/api/v1/system/health' and method == 'GET':
        health_data = {
            "status": "healthy",
            "timestamp": datetime.now().isoformat(),
            "components": {
                "api_gateway": "healthy",
                "auth_service": "healthy",
                "kafka": "healthy",
                "redis": "healthy",
                "postgres": "healthy",
                "clickhouse": "healthy"
            },
            "version": "1.0.0-demo"
        }
        return DemoResponse(200, health_data)
    
    elif endpoint == '/api/v1/system/metrics' and method == 'GET':
        metrics_data = {
            "goroutines": 150,
            "memory_usage_mb": 245.7,
            "active_connections": 23,
            "requests_per_second": 12.5,
            "uptime_seconds": 86400,
            "kafka_messages_processed": 12500
        }
        return DemoResponse(200, metrics_data)
    
    # По умолчанию - успешный ответ
    return DemoResponse(200, {})

def login_user(email, password):
    """Аутентификация пользователя"""
    login_data = {
        "email": email,
        "password": password
    }
    
    response = make_authenticated_request('POST', '/api/v1/auth/login', json=login_data)
    
    if response and response.status_code == 200:
        data = response.json()
        st.session_state.access_token = data['access_token']
        st.session_state.user_info = data['user']
        st.success("Успешный вход!")
        time.sleep(1)
        st.rerun()
    elif response:
        st.error(f"Ошибка входа: {response.json().get('message', 'Неизвестная ошибка')}")

def register_user(user_data):
    """Регистрация нового пользователя"""
    response = make_authenticated_request('POST', '/api/v1/auth/register', json=user_data)
    
    if response and response.status_code == 201:
        data = response.json()
        st.session_state.access_token = data['access_token']
        st.session_state.user_info = data['user']
        st.success("Регистрация успешна!")
        time.sleep(1)
        st.rerun()
    elif response:
        st.error(f"Ошибка регистрации: {response.json().get('message', 'Неизвестная ошибка')}")

def logout_user():
    """Выход из системы"""
    response = make_authenticated_request('POST', '/api/v1/auth/logout')
    st.session_state.access_token = None
    st.session_state.user_info = None
    st.success("Вы вышли из системы")
    time.sleep(1)
    st.rerun()

# ==================== СТРАНИЦА АУТЕНТИФИКАЦИИ ====================
def auth_page():
    st.title("🏭 Moscow Industrial Data Platform")
    
    # Баннер демо-режима
    if st.session_state.demo_mode:
        st.warning("🔶 ДЕМО-РЕЖИМ: Используются демо-данные. Бэкенд не требуется.")
    
    st.markdown("---")
    
    tab1, tab2 = st.tabs(["🔐 Вход", "📝 Регистрация"])
    
    with tab1:
        st.header("Вход в систему")
        
        # Демо-аккаунты для быстрого входа
        if st.session_state.demo_mode:
            st.info("**Демо-аккаунты:**")
            col1, col2 = st.columns(2)
            with col1:
                if st.button("👨‍💼 Войти как Компания", use_container_width=True):
                    login_user("company@demo.ru", "demo123")
            with col2:
                if st.button("📊 Войти как Аналитик", use_container_width=True):
                    login_user("analyst@demo.ru", "demo123")
            st.markdown("---")
        
        with st.form("login_form"):
            email = st.text_input("Email", placeholder="company@industrial.ru")
            password = st.text_input("Пароль", type="password", placeholder="Введите пароль")
            submit = st.form_submit_button("Войти")
            
            if submit:
                if email and password:
                    login_user(email, password)
                else:
                    st.error("Заполните все поля")
    
    with tab2:
        st.header("Регистрация новой компании")
        
        with st.form("register_form"):
            col1, col2 = st.columns(2)
            
            with col1:
                email = st.text_input("Email*", placeholder="company@industrial.ru")
                password = st.text_input("Пароль*", type="password", 
                                       help="Минимум 8 символов")
                company_name = st.text_input("Название компании*", 
                                           placeholder="ООО 'Промышленный завод Москвы'")
            
            with col2:
                role = st.selectbox("Роль*", ["company", "analyst"], 
                                  format_func=lambda x: "Компания" if x == "company" else "Аналитик")
                contact_person = st.text_input("Контактное лицо", 
                                             placeholder="Иванов Иван Иванович")
                phone = st.text_input("Телефон", placeholder="+7 495 123-45-67")
            
            submit = st.form_submit_button("Зарегистрироваться")
            
            if submit:
                if all([email, password, company_name, role]):
                    register_data = {
                        "email": email,
                        "password": password,
                        "role": role,
                        "company_name": company_name,
                        "contact_person": contact_person or "",
                        "phone": phone or ""
                    }
                    register_user(register_data)
                else:
                    st.error("Заполните обязательные поля (отмечены *)")

# ==================== ГЛАВНАЯ СТРАНИЦА ====================
def main_page():
    # Sidebar с информацией о пользователе
    with st.sidebar:
        st.header(f"👋 {st.session_state.user_info.get('company_name', 'Пользователь')}")
        st.write(f"**Email:** {st.session_state.user_info.get('email')}")
        st.write(f"**Роль:** {st.session_state.user_info.get('role')}")
        st.write(f"**Регистрация:** {st.session_state.user_info.get('created_at', '')[:10]}")
        
        if st.session_state.demo_mode:
            st.success("🔶 Демо-режим")
        
        if st.button("🚪 Выйти", use_container_width=True):
            logout_user()
        
        st.markdown("---")
        
        # Настройки API
        st.subheader("Настройки")
        
        demo_status = st.checkbox("Демо-режим", value=st.session_state.demo_mode)
        if demo_status != st.session_state.demo_mode:
            st.session_state.demo_mode = demo_status
            st.rerun()
        
        if not st.session_state.demo_mode:
            api_server = st.selectbox(
                "Сервер API",
                ["http://localhost:8080", "https://staging-api.mosprom-hackathon.ru", "https://api.mosprom-hackathon.ru"],
                index=0
            )
    
    # Основной контент
    st.title(f"🏭 {st.session_state.user_info.get('company_name', 'Industrial Platform')}")
    
    # Навигация по разделам
    tabs = st.tabs(["📊 Дашборд", "📁 Управление файлами", "📈 Аналитика", "👤 Профиль", "⚙️ Система"])
    
    # ==================== ВКЛАДКА ДАШБОРДА ====================
    with tabs[0]:
        st.header("Дашборд")
        
        col1, col2, col3 = st.columns(3)
        
        with col1:
            # Статистика файлов
            response = make_authenticated_request('GET', '/api/v1/files?limit=100')
            if response and response.status_code == 200:
                files_data = response.json()
                total_files = len(files_data.get('files', []))
                processed_files = len([f for f in files_data.get('files', []) if f.get('status') == 'processed'])
                
                st.metric("Всего файлов", total_files)
                st.metric("Обработано", processed_files)
        
        with col2:
            # Статистика отчетов
            response = make_authenticated_request('GET', '/api/v1/analytics/reports?limit=50')
            if response and response.status_code == 200:
                reports_data = response.json()
                total_reports = len(reports_data.get('reports', []))
                st.metric("Сгенерировано отчетов", total_reports)
        
        with col3:
            # Статус системы
            response = make_authenticated_request('GET', '/api/v1/system/health')
            if response and response.status_code == 200:
                health_data = response.json()
                status_color = "🟢" if health_data.get('status') == 'healthy' else "🔴"
                st.metric("Статус системы", f"{status_color} {health_data.get('status')}")
        
        # Последние файлы
        st.subheader("Последние файлы")
        response = make_authenticated_request('GET', '/api/v1/files?limit=5')
        if response and response.status_code == 200:
            files_data = response.json()
            if files_data.get('files'):
                for file in files_data['files']:
                    status_emoji = {
                        'uploaded': '📤',
                        'processing': '🔄',
                        'processed': '✅',
                        'failed': '❌'
                    }.get(file.get('status'), '📄')
                    
                    col1, col2, col3 = st.columns([3, 2, 2])
                    with col1:
                        st.write(f"{status_emoji} **{file.get('filename')}**")
                    with col2:
                        st.write(f"Тип: {file.get('data_type', 'N/A')}")
                    with col3:
                        st.write(f"Загружен: {file.get('created_at', '')[:16]}")
            else:
                st.info("Файлы еще не загружены")
    
    # ==================== ВКЛАДКА УПРАВЛЕНИЯ ФАЙЛАМИ ====================
    with tabs[1]:
        st.header("Управление файлами")
        
        col1, col2 = st.columns([1, 2])
        
        with col1:
            st.subheader("Загрузка файла")
            
            with st.form("upload_form"):
                uploaded_file = st.file_uploader(
                    "Выберите файл", 
                    type=['csv', 'xlsx', 'json'],
                    help="Поддерживаемые форматы: CSV, XLSX, JSON"
                )
                
                data_type = st.selectbox(
                    "Тип данных",
                    ["production", "financial", "energy", "environmental", "personnel"],
                    format_func=lambda x: {
                        "production": "Производственные",
                        "financial": "Финансовые", 
                        "energy": "Энергетические",
                        "environmental": "Экологические",
                        "personnel": "Кадровые"
                    }[x]
                )
                
                description = st.text_area("Описание файла")
                
                submit = st.form_submit_button("📤 Загрузить файл")
                
                if submit and uploaded_file is not None:
                    files = {'file': (uploaded_file.name, uploaded_file.getvalue())}
                    data = {
                        'description': description,
                        'data_type': data_type
                    }
                    
                    response = make_authenticated_request(
                        'POST', 
                        '/api/v1/files/upload', 
                        files=files,
                        data=data
                    )
                    
                    if response and response.status_code == 202:
                        st.success("Файл принят в обработку!")
                        st.rerun()
                    elif response:
                        st.error(f"Ошибка загрузки: {response.json().get('message')}")
        
        with col2:
            st.subheader("Мои файлы")
            
            # Фильтры
            col_f1, col_f2 = st.columns(2)
            with col_f1:
                status_filter = st.selectbox(
                    "Статус",
                    ["all", "uploaded", "processing", "processed", "failed"],
                    format_func=lambda x: "Все" if x == "all" else {
                        "uploaded": "Загружено",
                        "processing": "В обработке", 
                        "processed": "Обработано",
                        "failed": "Ошибка"
                    }[x]
                )
            
            # Получение списка файлов
            params = {}
            if status_filter != "all":
                params['status'] = status_filter
                
            response = make_authenticated_request('GET', '/api/v1/files', params=params)
            
            if response and response.status_code == 200:
                files_data = response.json()
                
                if files_data.get('files'):
                    for file in files_data['files']:
                        with st.expander(f"📄 {file.get('filename')}"):
                            col1, col2, col3 = st.columns(3)
                            
                            with col1:
                                st.write(f"**Размер:** {file.get('size', 0) / 1024:.1f} KB")
                                st.write(f"**Тип:** {file.get('data_type', 'N/A')}")
                            
                            with col2:
                                status_emoji = {
                                    'uploaded': '📤',
                                    'processing': '🔄', 
                                    'processed': '✅',
                                    'failed': '❌'
                                }.get(file.get('status'), '📄')
                                st.write(f"**Статус:** {status_emoji} {file.get('status')}")
                                st.write(f"**Загружен:** {file.get('created_at', '')[:16]}")
                            
                            with col3:
                                if file.get('status') == 'processed':
                                    st.success("Готово")
                                elif file.get('status') == 'processing':
                                    st.warning("В обработке")
                                elif file.get('status') == 'failed':
                                    st.error("Ошибка")
                                
                                # Кнопка удаления
                                if st.button("🗑️ Удалить", key=f"delete_{file.get('file_id')}"):
                                    delete_response = make_authenticated_request(
                                        'DELETE', 
                                        f"/api/v1/files/{file.get('file_id')}"
                                    )
                                    if delete_response and delete_response.status_code == 204:
                                        st.success("Файл удален")
                                        st.rerun()
                else:
                    st.info("Файлы не найдены")
    
    # ==================== ВКЛАДКА АНАЛИТИКИ ====================
    with tabs[2]:
        st.header("Аналитика и отчеты")
        
        tab_analytics1, tab_analytics2 = st.tabs(["📋 Мои отчеты", "🔄 Генерация отчета"])
        
        with tab_analytics1:
            st.subheader("История отчетов")
            
            response = make_authenticated_request('GET', '/api/v1/analytics/reports?limit=20')
            if response and response.status_code == 200:
                reports_data = response.json()
                
                if reports_data.get('reports'):
                    for report in reports_data['reports']:
                        with st.expander(f"📊 {report.get('report_name', 'Отчет')}"):
                            col1, col2 = st.columns(2)
                            with col1:
                                st.write(f"**Тип:** {report.get('report_type')}")
                                st.write(f"**Сгенерирован:** {report.get('generated_at', '')[:16]}")
                            with col2:
                                st.write(f"**Размер:** {report.get('file_size', 0) / 1024:.1f} KB")
                                if report.get('download_url'):
                                    st.download_button(
                                        "📥 Скачать отчет",
                                        data=b"demo content",
                                        file_name=f"{report.get('report_name', 'report')}.pdf",
                                        key=f"download_{report.get('report_id')}"
                                    )
                else:
                    st.info("Отчеты еще не генерировались")
        
        with tab_analytics2:
            st.subheader("Создание нового отчета")
            
            with st.form("generate_report"):
                col1, col2 = st.columns(2)
                
                with col1:
                    report_type = st.selectbox(
                        "Тип отчета*",
                        ["production", "financial", "energy", "environmental", "comprehensive"],
                        format_func=lambda x: {
                            "production": "Производственный",
                            "financial": "Финансовый",
                            "energy": "Энергетический", 
                            "environmental": "Экологический",
                            "comprehensive": "Комплексный"
                        }[x]
                    )
                    
                    format_type = st.selectbox(
                        "Формат*",
                        ["pdf", "excel", "json"],
                        format_func=lambda x: x.upper()
                    )
                
                with col2:
                    start_date = st.date_input("Начальная дата*")
                    end_date = st.date_input("Конечная дата*")
                
                filters = st.text_area("Дополнительные фильтры (JSON)", 
                                     help='Пример: {"department": "production"}')
                
                submit = st.form_submit_button("🔄 Сгенерировать отчет")
                
                if submit:
                    if start_date and end_date:
                        report_data = {
                            "report_type": report_type,
                            "time_range": {
                                "start_date": str(start_date),
                                "end_date": str(end_date)
                            },
                            "format": format_type
                        }
                        
                        if filters:
                            try:
                                report_data["filters"] = json.loads(filters)
                            except:
                                st.error("Неверный формат JSON в фильтрах")
                                return
                        
                        response = make_authenticated_request(
                            'POST', 
                            '/api/v1/analytics/reports/generate',
                            json=report_data
                        )
                        
                        if response and response.status_code == 202:
                            st.success("Отчет поставлен в очередь на генерацию!")
                            st.rerun()
                        elif response:
                            st.error(f"Ошибка: {response.json().get('message')}")
                    else:
                        st.error("Заполните обязательные поля")
    
    # ==================== ВКЛАДКА ПРОФИЛЯ ====================
    with tabs[3]:
        st.header("Профиль пользователя")
        
        # Получение актуальной информации
        response = make_authenticated_request('GET', '/api/v1/users/me')
        if response and response.status_code == 200:
            user_data = response.json()
            
            col1, col2 = st.columns(2)
            
            with col1:
                st.subheader("Текущая информация")
                st.write(f"**ID:** {user_data.get('id')}")
                st.write(f"**Email:** {user_data.get('email')}")
                st.write(f"**Роль:** {user_data.get('role')}")
                st.write(f"**Компания:** {user_data.get('company_name')}")
                st.write(f"**Контактное лицо:** {user_data.get('contact_person')}")
                st.write(f"**Телефон:** {user_data.get('phone')}")
                st.write(f"**Регистрация:** {user_data.get('created_at', '')[:16]}")
            
            with col2:
                st.subheader("Обновление профиля")
                
                with st.form("update_profile"):
                    new_company = st.text_input("Название компании", value=user_data.get('company_name', ''))
                    new_contact = st.text_input("Контактное лицо", value=user_data.get('contact_person', ''))
                    new_phone = st.text_input("Телефон", value=user_data.get('phone', ''))
                    
                    submit = st.form_submit_button("💾 Сохранить изменения")
                    
                    if submit:
                        update_data = {}
                        if new_company != user_data.get('company_name'):
                            update_data['company_name'] = new_company
                        if new_contact != user_data.get('contact_person'):
                            update_data['contact_person'] = new_contact
                        if new_phone != user_data.get('phone'):
                            update_data['phone'] = new_phone
                        
                        if update_data:
                            response = make_authenticated_request(
                                'PUT', 
                                '/api/v1/users/profile',
                                json=update_data
                            )
                            
                            if response and response.status_code == 200:
                                st.success("Профиль обновлен!")
                                st.session_state.user_info = response.json()
                                st.rerun()
                            elif response:
                                st.error(f"Ошибка обновления: {response.json().get('message')}")
    
    # ==================== ВКЛАДКА СИСТЕМЫ ====================
    with tabs[4]:
        st.header("Системная информация")
        
        col1, col2 = st.columns(2)
        
        with col1:
            st.subheader("Статус системы")
            response = make_authenticated_request('GET', '/api/v1/system/health')
            if response and response.status_code == 200:
                health_data = response.json()
                
                status_color = {
                    'healthy': '🟢',
                    'degraded': '🟡', 
                    'unhealthy': '🔴'
                }.get(health_data.get('status'), '⚪')
                
                st.metric("Общий статус", f"{status_color} {health_data.get('status')}")
                st.write(f"**Время проверки:** {health_data.get('timestamp', '')[:16]}")
                st.write(f"**Версия API:** {health_data.get('version')}")
                
                st.subheader("Компоненты")
                for component, status in health_data.get('components', {}).items():
                    comp_color = '🟢' if status == 'healthy' else '🔴'
                    st.write(f"{comp_color} {component}: {status}")
        
        with col2:
            st.subheader("Метрики")
            response = make_authenticated_request('GET', '/api/v1/system/metrics')
            if response and response.status_code == 200:
                metrics_data = response.json()
                
                st.metric("Активные горутины", metrics_data.get('goroutines', 0))
                st.metric("Использование памяти (MB)", f"{metrics_data.get('memory_usage_mb', 0):.1f}")
                st.metric("Активные соединения", metrics_data.get('active_connections', 0))
                st.metric("Запросов в секунду", f"{metrics_data.get('requests_per_second', 0):.1f}")
                st.metric("Время работы (часы)", f"{metrics_data.get('uptime_seconds', 0) / 3600:.1f}")

# ==================== ОСНОВНАЯ ЛОГИКА ====================
def main():
    # Настройка API через sidebar (даже для неавторизованных)
    with st.sidebar:
        st.image("https://via.placeholder.com/150x50/0047AB/FFFFFF?text=Mosprom", width=150)
        st.markdown("---")
    
    # Проверка авторизации
    if st.session_state.access_token and st.session_state.user_info:
        main_page()
    else:
        auth_page()

if __name__ == "__main__":
    main()
