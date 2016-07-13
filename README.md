# Calicoctl
This repository contains a GO version of the calicoctl binary and the libcalico library.

## Building calicoctl
Assuming you have already installed **go v1.6+**, perform the following simple steps to get building:

1. [Install Glide](https://github.com/Masterminds/glide#install)

2. Clone this repository to your Go project path: 
    ```
    git clone git@github.com:projectcalico/calicoctl.git $GOPATH/src/github.com/projectcalico/calicoctl
    ``` 

3. Switch to your project directory:
    ```
    cd $GOPATH/src/github.com/projectcalico/calicoctl
    ```

4. Populate the `vendor/` directory in the project's root with this project's dependencies:
    ```
    glide install
    ```

5. Build calicoctl:
    ```
    go build -o calicoctl calicoctl.go
    ```

6. Run your built binary:
    ```
    ./calicoctl --help
    ```



A dockerized build of calicoctl is available which builds calicoctl in a CentOS 6.6 container.

Run:
    ```
    ./build-binary.sh
    ```