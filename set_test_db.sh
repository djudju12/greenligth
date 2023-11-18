#!/bin/bash

# the porpuse of this script is to create a containerize db to run
# all integration tests. This is not a well spoken technique, but
# something that I think its gonna work...
source .cfg

DB_USERNAME=postgres
DB_PASSWORD=postgres

#
# util so we can easy clean up
#
clean_up() {
   echo 'stoping and removing any test dbs containers left over'
   docker container stop $CONTAINER_NAME_TEST
   docker container rm $CONTAINER_NAME_TEST
}

echo 'creating container...'
until docker run --name $CONTAINER_NAME_TEST        \
           -p $DB_EXTERNALPORT_TEST:$DB_PORT        \
           --tmpfs /var/lib/postgresql/data:rw \
           -e POSTGRES_PASSWORD=$DB_PASSWORD   \
           -d $IMAGE_DB
do
   clean_up
done

echo 'start checking to se if the db is ready...'
timeout 25s bash -c "until docker exec ${CONTAINER_NAME_TEST} pg_isready; do sleep 5; done"
exit_status=$?
if [[ exit_status -ne 0 ]]; then
   echo 'unable to connect with test db. Exiting script...'
   clean_up
   exit 1
fi

DB_DSN=postgres://$DB_USERNAME:$DB_PASSWORD@localhost:$DB_EXTERNALPORT_TEST
DB_NAME=test_db

echo "creating test db: $DB_NAME"
PGPASSWORD=$DB_PASSWORD psql --host=localhost                       \
                             --port=$DB_EXTERNALPORT_TEST           \
                             --username=$DB_USERNAME                \
                             --command="create database ${DB_NAME}" \

echo "creating extensions"
PGPASSWORD=$DB_PASSWORD psql --host=localhost                                  \
                             --port=$DB_EXTERNALPORT_TEST                      \
                             --username=$DB_USERNAME                           \
                             -d $DB_NAME                                       \
                             --command="CREATE EXTENSION IF NOT EXISTS citext" \


DB_DSN=$DB_DSN/$DB_NAME?sslmode=disable

echo 'making migrations...'
MIGRATION_PAH=./migrations
migrate -path $MIGRATION_PAH -database $DB_DSN up
exit_status=$?
if [[ exit_status -ne 0 ]]; then
   echo 'failed migrations...'
   clean_up
   exit 1
fi

echo 'all set!'

echo $DB_DSN