# Integration Test Environment

Docker Compose runs the web application with a shared SQLite database and loads realistic fixture data.

## Quick start

```sh
cd test/integration
docker compose up --build
```

The UI is available at `http://localhost:9000/`. Set `HOST_PORT` to use a different host port.

The one-shot `seed` service runs `quantriskcli seed`, creating an automation user and loading scenarios, controls, requirements, gaps, FAIR-CAM assignments, and relationships.

## Tear down

```sh
docker compose down -v
```
