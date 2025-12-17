FROM python:3.11-slim

# Avoid interactive prompts
ENV DEBIAN_FRONTEND=noninteractive
ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

WORKDIR /app

# System deps (minimal)
RUN apt-get update && apt-get install -y \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Python deps
COPY ml_service/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application
COPY ml_service/ .

# Expose FastAPI port
EXPOSE 8000

# Start service
CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8000"]
