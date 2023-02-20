FROM python:3.11.2-slim-bullseye

RUN pip install PyYAML

COPY github-workflows-validator.py /docker-entrypoint.py
RUN chmod +x /docker-entrypoint.py

ENV DOT_GITHUB_PATH ""

ENTRYPOINT ["python", "/docker-entrypoint.py"]
