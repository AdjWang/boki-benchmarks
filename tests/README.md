## How to add a new app

Take `tests/workloads/sharedlog` as an example.

1. create a new app named as "newapp"
2. add a new build command in `tests/workloads/build_all.sh`
3. modify its dockerfile at `tests/dockerfiles/`
4. modify its function info and add the corresponding test case parameter in `tests/docker-compose-generator.py`
5. modify its test case function in `tests/test_all.sh`
