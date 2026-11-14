FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1
WORKDIR /srv

COPY pyproject.toml ./
RUN pip install --no-cache-dir "fastapi>=0.115" "uvicorn[standard]>=0.30" "sqlglot>=30,<31"

COPY app ./app
RUN useradd --system --no-create-home bi && chown -R bi /srv
USER bi

EXPOSE 8000
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
