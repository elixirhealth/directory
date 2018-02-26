# Directory
Directory service manages entity information and is usually backed by a Postgres DB. 

### Migrations

When you need to make changes to the Postgres DDL, add a new migration `up` and `down` SQL files to 
`pkg/server/storage/postgres/migrations/sql` and then run 
```bash
make migrations
```
which will use [go-bindata](https://github.com/jteeuwen/go-bindata) to bundle this SQL into the 
`pkg/server/storage/migrations` package.

### Running tests

Tests in the `pkg/server/storage/postgres` package depend on a running Postgres DB. Running these 
tests from an IDE (e.g., Goland) therefore will not work if you do not have Postgres installed on 
your local dev machine. The recommended way to run these is within the build container (which is 
what CircleCI does). You can do this on your local machine via
```bash
$ make enter-build-container
# enter "elxir-build" host 
$ cd src/github.com/elxirhealth/directory
$ make test
```

The Postgres tests in [postgres/storer_test.go](pkg/server/storage/postgres/storer_test.go) start a 
local Postgres DB and apply the `up` migrations before and `down` migrations after each test that 
needs them. 

### Augmenting entity type definitions

Adding a new entity type:
1) In [directory.proto](pkg/directoryapi/directory.proto), add new message type and add to 
`type_attributes` in the `Entity` message type.
2) Increment `NEntityTypes` in [entity_type.go](pkg/server/storage/entity_type.go) and add to the 
entityType enum values.
3) Create a migration adding a table for the new type.
4) Run the tests, and they will tell you what you need to fix (usually by adding a case for the
new type in functions in [entity_type.go](pkg/server/storage/entity_type.go) and in 
[postgres/entity.go](pkg/server/storage/postgres/entity.go).

Adding fields to an existing entity type:
1) Add new fields to the appropriate existing message type (additive changes only) and run 
`make proto`.
2) Add new fields to the appropriate `prep*Scan` and `getPut*Values` functions in 
[postgres/entity.go](pkg/server/storage/postgres/entity.go) and update appropriate tests. 
 


