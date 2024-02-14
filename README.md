# mpn-server

## Container

```
podman pull quay.io/simple64/mpn-server:latest
```

## Running
```
./mpn-server --name "Server Name"
```
## Running with Podman
```
cat << EOF > pod.yaml
apiVersion: v1
kind: Pod
metadata:
  name: netplay_pod
  namespace: netplay
spec:
  hostNetwork: true
  containers:
    - name: netplay_container
      image: quay.io/simple64/mpn-server:latest
      args:
        - "--name"
        - "My Server"
        - "--disable-broadcast"
        - "--baseport"
        - "45000"
EOF

podman play kube pod.yaml
```
## Playing locally
The server is discoverable on a LAN. When the server is running, clients on the same LAN should find the server automatically.

## Port/firewall requirements
The server will be listening on ports 45000-45010 by default, using TCP and UDP. Firewalls will need to be configured to allow connections on these ports.
