FROM postgres:17-alpine

COPY migrations /migrations

RUN sed -i 's/\r$//' /migrations/run.sh /migrations/*.sql \
    && chmod +x /migrations/run.sh

ENTRYPOINT ["/bin/sh", "/migrations/run.sh"]
