.PHONEY: all test ut update-vendor

default: all
all: test
test: ut

update-vendor:
	glide up

ut:
	./run-uts

.PHONEY: go-rebuild
go-rebuild:
	true
bin/etcd-driver: go-rebuild
	mkdir -p bin
	go build -o "$@" "./$(@F)/..."

bin/calicoctl: go-rebuild
	mkdir -p bin
	go build -o "$@" "./calicoctl/calicoctl.go"

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
