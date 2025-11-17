# TestTask-17.11.2025
Сервис для проверки доступности ссылок на интернет-ресурсы

## Routes
| Метод   | Route                | Описание                           |
|---------|----------------------|------------------------------------|
| POST    | `/check`             | Проверка доступности ссылок        |
| POST    | `/list`              | Отчёт в формате pdf наборов ссылок |


## Пример использования

Ипользуйте следующую команду для копирования репозитория.
```bash
git clone https://github.com/MimPe9/TestTask-17.11.2025
```

Запрос на проверку доступности ссылок в наборе:
```bash
body1='{"links": ["google.com", "youtube.com"]}'
curl -X POST http://localhost:8020/check \
  -H "Content-Type: application/json" \
  -d "$body1"
```
Запрос на отчёт в формате pdf для интервала наборов:
```bash
pdf_request='{"nums": [1, 2]}'
curl -X POST http://localhost:8020/list \
  -H "Content-Type: application/json" \
  -d "$pdf_request" \
  --output report.pdf
```
