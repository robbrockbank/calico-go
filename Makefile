.PHONEY: all test ut update-vendor

default: all
all: test
test: ut

update-vendor:
	glide up

ut:
	./run-uts

.PHONEY: force
force:
	true
bin/etcd-driver: force
	mkdir -p bin
	go build -o "$@" "./$(@F)/..."

bin/calicoctl: force
	mkdir -p bin
	go build -o "$@" "./calicoctl/calicoctl.go"

release/calicoctl: force
	mkdir -p release
	docker build -t calicoctl-build -f build-calicoctl/Dockerfile .
	docker run --rm -v `pwd`:/calico-go calicoctl-build /calico-go/build-calicoctl/build.sh

clean:
	-rm -f *.created
	find . -name '*.pyc' -exec rm -f {} +
	-rm -rf build
	-rm -rf calico_containers/pycalico.egg-info/
	-docker rm -f calico-build
	-docker rmi calico/build

setup-env:
	virtualenv venv
	venv/bin/pip install --upgrade -r requirements.txt
	@echo "run\n. venv/bin/activate"
