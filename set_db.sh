#!/bin/bash

source .cfg
source .env

#
# util so we can easy clean up
#
clean_up() {
   >&2 echo 'stoping and removing any test dbs containers left over'
   docker container stop $CONTAINER_NAME
   docker container rm $CONTAINER_NAME
}

>&2 echo 'creating container...'
until docker compose up -d
do
   clean_up
done

>&2 echo 'start checking to se if the db is ready...'
timeout 25s bash -c "until docker exec ${CONTAINER_NAME} pg_isready; do sleep 5; done"
exit_status=$?
if [[ exit_status -ne 0 ]]; then
   echo 'unable to connect with test db. Exiting script...'
   clean_up
   exit 1
fi

DB_DSN=postgres://$DB_USERNAME:$DB_PASSWORD@localhost:$DB_PORT

>&2 echo "creating db: $DB_NAME"
PGPASSWORD=$DB_PASSWORD psql --host=localhost --port=$DB_PORT --username=$DB_USERNAME \
                             --command="CREATE DATABASE ${DB_NAME}"                   \

>&2 echo "creating extensions"
PGPASSWORD=$DB_PASSWORD psql --host=localhost --port=$DB_PORT --username=$DB_USERNAME -d $DB_NAME \
                             --command="CREATE EXTENSION IF NOT EXISTS citext"                    \

>&2 echo "creating default user"
read -p "Enter password for greenlight DB user: " DB_USER_PASSWORD
PGPASSWORD=$DB_PASSWORD psql --host=localhost --port=$DB_PORT -d $DB_NAME --username=$DB_USERNAME         \
                             --command="CREATE ROLE ${DB_NAME} WITH LOGIN PASSWORD '${DB_USER_PASSWORD}'" \


echo "" >> .env
echo "GREENLIGHT_DB_DSN='postgres://${DB_NAME}:${DB_USER_PASSWORD}@localhost:${DB_PORT}?sslmode=disable'" >> .env

>&2 echo 'all set!'