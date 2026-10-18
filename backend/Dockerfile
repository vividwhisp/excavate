FROM python:3.12-slim AS build
WORKDIR /app
COPY pyproject.toml ./
RUN pip install --no-cache-dir fastapi uvicorn asyncpg httpx bcrypt pydantic-settings beautifulsoup4 lxml
COPY app ./app
COPY app/store/migrations ./app/store/migrations

FROM python:3.12-slim
RUN adduser --disabled-password --gecos "" appuser
WORKDIR /app
COPY --from=build /usr/local/lib/python3.12/site-packages /usr/local/lib/python3.12/site-packages
COPY --from=build /usr/local/bin /usr/local/bin
COPY --from=build /app /app
USER appuser
EXPOSE 8080
ENV PYTHONUNBUFFERED=1
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8080"]
