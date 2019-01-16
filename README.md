# Controller for Kubernetes Custom Resource

`artifacts/crd.crd.yaml` contains the spec of the crd, and `crdob.crdob.yaml` contains spec of an object of `Jackpot` kind

apply ```kubectl create -f artifacts/``` to create the crd and Jackpot

Build the Go Controller with
```
go build
```
Run `./booklist-controller` to start the controller

Then if you change `Image`, `Replicas` of Jackpot, you will see the corresponding change in the `booklist-controller` Deployment object
