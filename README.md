# Project HireMeMaybe-backend

Backend service for HireMeMaybe web application platform for employment focused on Computer Engineering and Software and Knowledge Engineering student at Kasetsart university

## Requirement
- golang  v1.24.6
- docker

## How to run
- Copy sample.env and fill Google client ID and Secret other you can leave it as default

Mac / Linux:
```
cp ./sample.env .env
```

Window:
```
copy ./sample.env .env
```

- Start postgres docker container 
```
make docker run
```

- Install psql and run this command
```
PGPASSWORD=<database password> psql -h 127.0.0.1 -U <database username> -d <database name> -f ./init_extension.sql
```

- Install dependencies
```
go get .
```

- Run server
```
make run
```

## MakeFile

Run build make command with tests
```bash
make all
```

Build the application
```bash
make build
```

Run the application
```bash
make run
```
Create DB container
```bash
make docker-run
```

Shutdown DB Container
```bash
make docker-down
```

DB Integrations Test:
```bash
make itest
```

Live reload the application:
```bash
make watch
```

Run the test suite:
```bash
make test
```

Clean up binary from the last build:
```bash
make clean
```
