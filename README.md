# Pihole Linktree

![Alt text](pihole-linktree.png?raw=true "Screen Shot")

A simple web application that displays a list of domains and their corresponding IP addresses from a Pi-hole DNS server. Built with Go 1.23.4.

## Features

- Fetches custom DNS records from a Pi-hole server via the API
- Displays domains as clickable HTTPS links with their corresponding IP addresses
- Clean, responsive web interface with modern styling using Bootstrap 5
- Containerized deployment using Docker and Alpine Linux
- Automatic health checks and restart on failure
- DNS resolution through Pi-hole
- 15-second cache refresh interval
- Title extraction from linked domains

## Prerequisites

- Docker Engine 24.0+ and Docker Compose v2+
- A running Pi-hole instance with API access enabled
- Pi-hole API token (found in Settings > API > Show API token)

## Configuration

The service is configured using environment variables in the `.env` file:

- `PIHOLE_HOST`: Hostname of your Pi-hole server
- `PIHOLE_TOKEN`: Pi-hole API token (64-character string)
- `BASE_DOMAIN`: Base domain for filtering records
- `PIHOLE_DNS`: IP address of Pi-hole DNS server for container DNS resolution
- `CACHE_REFRESH_INTERVAL`: Refresh interval in seconds

The service runs on port 8080 and requires a fixed IP address.

## Building and Running

1. Clone the repository:

   ```bash
   git clone https://github.com/jturlington/pihole-linktree.git
   cd pihole-linktree
   ```

2. Create a `.env` file with your configuration:

   ```env
   PIHOLE_HOST= # Pi-hole hostname
   PIHOLE_TOKEN= # Pi-hole API token (found in Settings > API  > Show API token)
   BASE_DOMAIN= # Base domain for filtering records
   PIHOLE_DNS= # Pi-hole DNS server IP
   CACHE_REFRESH_INTERVAL= # Refresh interval in seconds
   ```

3. Build and start the container:

   ```bash
   # Build the container
   docker compose build

   # Start in detached mode
   docker compose up -d

   # View logs
   docker compose logs -f
   ```

# Why I wrote it?

Ever feel like you're playing a game of hide-and-seek with your own applications? Yeah, me too. It was driving me nuts! I'd spin up some cool new service, give it some cutesy, totally-forgettable name like "FluffyBunnyServer" – or maybe something even more ridiculous, like "CosmicBurrito.jont.us" – because, why not, right?  And then, a week later, I'd need to tweak something. Where was it? What IP did I assign it? Was it even still running? I was constantly digging through old terminal windows, sifting through config files, trying to piece together the digital breadcrumbs I'd left for myself. It was a nightmare! I wasted so much time just finding my stuff that I barely had any left to actually work on it. So, I built this. This little app right here? It's my sanity saver. It's the answer to my "where did I put that?!" prayers. No more cryptic names, no more IP address scavenger hunts, even with domains like jont.us thrown in the mix. Just a simple, clean interface that shows me everything I've got running, where it is, and what it's doing. Consider it my digital decluttering project. And honestly? It's been a game changer.
