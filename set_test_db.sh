#!/bin/bash

# the porpuse of this script is to create a containerize db to run
# all integration tests. This is not a well spoken technique, but
# something that I think its gonna work...

#
# NOTE: >&2 means that we are redirecting logs do stderr
#
source .cfg

DB_USERNAME=postgres
DB_PASSWORD=postgres

#
# util so we can easy clean up
#
clean_up() {
   >&2 echo 'stoping and removing any test dbs containers left over'
   >&2 docker container stop $CONTAINER_NAME_TEST
   >&2 docker container rm $CONTAINER_NAME_TEST
}

>&2 echo 'creating container...'
until docker run --name $CONTAINER_NAME_TEST -q \
           -p $DB_EXTERNALPORT_TEST:$DB_PORT    \
           --tmpfs /var/lib/postgresql/data:rw  \
           -e POSTGRES_PASSWORD=$DB_PASSWORD    \
           -d $IMAGE_DB &> /dev/null
do
   clean_up
done

>&2 echo 'start checking to see if the db is ready...'
timeout 25s bash -c "until docker exec ${CONTAINER_NAME_TEST} pg_isready --quiet; do sleep 5; done"
exit_status=$?
if [[ exit_status -ne 0 ]]; then
   >&2 echo 'unable to connect with test db. Exiting script...'
   clean_up
   exit 1
fi

DB_DSN=postgres://$DB_USERNAME:$DB_PASSWORD@localhost:$DB_EXTERNALPORT_TEST
DB_NAME=test_db

>&2 echo "creating test db: $DB_NAME"
PGPASSWORD=$DB_PASSWORD psql --host=localhost -q                    \
                             --port=$DB_EXTERNALPORT_TEST           \
                             --username=$DB_USERNAME                \
                             --command="create database ${DB_NAME}" \
                             --quiet

>&2 echo "creating extensions"
PGPASSWORD=$DB_PASSWORD psql --host=localhost -q                               \
                             --port=$DB_EXTERNALPORT_TEST                      \
                             --username=$DB_USERNAME                           \
                             -d $DB_NAME                                       \
                             --command="CREATE EXTENSION IF NOT EXISTS citext" \
                             --quiet


DB_DSN=$DB_DSN/$DB_NAME?sslmode=disable

>&2 echo 'making migrations...'
MIGRATION_PAH=./migrations
migrate -path $MIGRATION_PAH -database $DB_DSN up
exit_status=$?
if [[ exit_status -ne 0 ]]; then
   >&2 echo 'failed migrations...'
   clean_up
   exit 1
fi

>&2 echo 'all set!'

>&1 echo $DB_DSN