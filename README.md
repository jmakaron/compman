# Compman 
### a company-management microservice

### API
* ```GET <host-ip>:<host-port>/company-manager/company``` \
  returns a list of JSON Objects of all the companies.
* ```GET <host-ip>:<host-port>/company-manager/company/<company-id>``` \
  returns a JSON Object of the company with the given id
* ```POST <host-ip>:<host-port>/company-manager/company/<company-id>``` \
  creates a new company, from the JSON Object in the body of the request
* ```PATCH <host-ip>:<host-port>/company-manager/company/<company-id>``` \
  updates company fields contained in the JSON Object in the body of the request, for company with the given id
* ```DELETE <host-ip>:<host-port>/company-manager/company/<company-id>``` \
  deletes company with the given id

#### Build
Simply, use the Makefile in the root project directory.
* ##### local binary
```$ make```
* ##### docker image
```$ make docker```
#### Configuration file
* #### local binary
./config/config.json can be used
* #### docker image 
./config/d_config.json can be used. The fields addr of JSON Object db and bootstrap_servers of JSON Object kp,
should be changed so they have the ip address of the host running docker compose.
#### Running Service & dependenies for local binary (postgres, kafka, zookeeper)
  ```./cloudbuild/$ docker compose up```
  Then run the binary 
  ```$ ./bin/compman -cfg=./config/config.json```
  or optionally in debug mode
  ```$ ./bin/compman -cfg=./config/config.json -debug```
#### Running Service & dependencies in docker (postgres, kafka, zookeeper)
  edit the ```./cloudbuild/docker-compose-with-compman.yml``` file
  and set ```KAFKA_ADVERTISED_LISTENERS``` to  ```PLAINTEXT://kafka:29092,PLAINTEXT_HOST://<host ip address>:9092```
  where ```<host ip address>``` is the ip address of the host running docker compose.
  Also edit the ```command``` field of compman to remove debug mode (by removing ```"--debug=true"``` field)
  Run ```./cloudbuild/$ docker compose up -f ./docker-compose-with-compman.yml```
  to run the compman service image and its dependencies
