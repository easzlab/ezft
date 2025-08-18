# Using EZFT to transfer files over SSH

## Scenario A:

Download files from a remote target server on a local computer; the remote SSH server and remote target server are located in the same network;

**Topology:**

```
[Local Computer] → [Remote SSH Server 192.168.0.1] → [Remote Target Server 192.168.0.2]
       |                                                                   | web server    
       |- - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -> 
```

**Setup:**

- Remote target server:

```
# start file server
./ezft server -p 18080
```

- Local computer:

```
# Enable SSH local port forwarding, assuming SSH server port is 10022, username is user
ssh -p 10022 -L 127.0.0.1:8080:192.168.0.2:18080 user@192.168.0.1 -N -f

# Download on local computer
./ezft client -u http://127.0.0.1:8080/path/to/file -c 4
```

## Scenario B:

Download files from a local computer on the remote ssh server;

**Topology:**

```
[Local Computer] → [Remote SSH Server 192.168.0.1] 
    | web server                      | 
    <- - - - - - - - - - - - - - - - -|
```

**Setup:**

- Local computer:

```
# Enable SSH remote port forwarding, assuming SSH server port is 10022, username is user
ssh -p10022 -R 127.0.0.1:18080:127.0.0.1:8080 user@192.168.0.1 -N -f

# start file server 
./ezft server -p 8080
```

- Remote ssh server:
```
# Download on remote ssh server
./ezft client -u http://127.0.0.1:18080/path/to/file -c 4
```