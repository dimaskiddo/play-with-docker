# Play With Docker

Play With Docker gives you the experience of having a free Alpine Linux Virtual Machine in the cloud
where you can build and run Docker containers and even create clusters with Docker features like Swarm Mode.

Under the hood DIND or Docker-in-Docker is used to give the effect of multiple VMs/PCs.

### Requirements

* [Docker](https://docs.docker.com/install/)
* [Go](https://golang.org/dl/)

### Deployment

```bash
# Clone this repo locally
git clone https://github.com/dimaskiddo/play-with-docker.git
cd play-with-docker

# Verify the Docker daemon is running
docker run hello-world

# Load the IPVS kernel module. Because swarms are created in dind,
# the daemon won't load it automatically
sudo modprobe xt_ipvs

# Ensure the Docker daemon is running in swarm mode
docker swarm init

# Get the latest franela/dind image
docker pull franela/dind:latest

# Start Play With Docker as a container
docker compose up -d
```

Now navigate to [http://localhost:3000](http://localhost:3000) and click the green "Start" button
to create a new session, followed by "ADD NEW INSTANCE" to launch a new terminal instance.

Notes:

* There is a hard-coded limit of 5 Docker playgrounds per session. After 4 hours sessions are deleted.
* If you want to override the DIND version or image then set the environmental variable `PWD_DEFAULT_DIND_IMAGE=franela/dind:latest` [franela](https://hub.docker.com/r/franela/).

### Port Forwarding

In order for port forwarding to work correctly in development you need to make `*.localhost` to resolve to `127.0.0.1`. That way when you try to access `pwd10-0-0-1-8080.host1.localhost`, then you're forwarded correctly to your local PWD server.

You can achieve this by setting up a `dnsmasq` server (you can run it in a docker container also) and adding the following configuration:

```
address=/localhost/127.0.0.1
```

Don't forget to change your computer's default DNS to use the dnsmasq server to resolve.


## FAQ

### How Can I Connect to a Published Port from The Outside World?

If you need to access your services from outside, use the following URL pattern `http://ip<hyphen-ip>-<session_jd>-<port>.<l2-subdomain>.<domain>` (i.e: http://ip10-10-10-10-b8ir6vbg5vr00095iil0-8080.apps.docker.dimaskiddo.my.id).

### Why is Play With Docker Running in Ports 80 and 443? Can I Change That?

No, it needs to run on those ports for DNS resolve to work. Ideas or suggestions about how to improve this are welcome

## Hints

### How can I use Copy / Paste Shortcuts?

- Ctrl  + Insert : Copy
- Shift + Insert : Paste
