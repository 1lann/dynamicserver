# dynamicserver
A system to dynamically launch DigitalOcean droplets as Minecraft servers if people are trying to connect, and creates a snapshot and destroys the droplet when there's nobody on the server to save running costs. A "launch on demand" model similar to [toffer/minecloud](https://github.com/toffer/minecloud), except instead of using a web browser, everything can be interacted with through the Minecraft multiplayer server list.

# Description
You can see the status of the server on the server list, and connect to it if it's not running to start it. It has an intelligent reverse proxy which automatically routes connections based on hostnames, essentially VirtualHosts for Minecraft allowing you to run and manage multiple severs routed through a single server/IP address. The reverse proxy can provide helpful error messages if issues occur, and is able to detect issues on its own and put up an "unavailability" warning for users trying to connect.

No web browser is required as it is controlled by people trying to connect, and configuration is done through .json files, so no database required either. Configuration is live reloaded and is automatically applied when they are changed, which allows for zero downtime modifications, even people who are already connected and playing on the server won't disconnect.

# Features
- Control and status viewing from Minecraft server list menu.
- Supports Minecraft 1.7 and up
- Supports a startup whitelist
- Configurable message headers
- Configurable droplet settings (Location, SSH key and droplet size)
- Works with any Minecraft server (FTB, Vanilla, Spigot, Tekkit, Cuberite, etc...)
- Tells users that the server is not available if it crashes, freezes, or is manually stopped.
- Can be forced into unavailability for maintenance purposes.
- Uses DigitalOcean's built in snapshot features to save and restore servers.
- Keeps last 3 snapshots, automatically deletes old ones. Can be used as a last resort backup.
- Supports different hostnames for different servers like virtual hosts for websites.
- Supports multiple hostnames per server.
- Routes people to connect to the back end servers.
- Is ruggedized and stateless. Will continue to work even in strange scenarios such as communication or an error with the backend.
- Will not attempt to shutdown the server if an issue is detected to prevent damage and for diagnostic purposes.

# Notice
Everyone's IP addresses will appear to be the same in the logs of the back end server due to the front end server acting as a reverse proxy. This means you effectively cannot IP ban players, unless you do so through the use of the front end server's firewall. Unfortunately at this time, people's IP addresses are not recorded anywhere.

# Setup
Grab the binaries from [releases](https://github.com/1lann/dynamicserver/releases), or run `go get github.com/1lann/dynamicserver/reverse_proxy` for the "front end" reverse proxy, or `go get github.com/1lann/dynamicserver/backend` for the "back end" helper to be installed in your `$GOPATH/bin` folder.

## Setting up the front end reverse proxy
The front end server is what people will connect to start the servers and route them, and is also manages the droplets running the back end servers. Note that the front end server should be running 24/7, and requires very little resources. You can run it on a droplet you already run 24/7, or use a 512 MB droplet which only costs $5/month.

1. Download the sample front end configuration from [here](https://github.com/1lann/dynamicserver/blob/master/reverse_proxy/config_sample.json).
2. Rename it to `config.json`.
3. Edit the configuration to fit your needs. Configuration documentation is available here.
4. Download the `reverse_proxy` front end software and make sure it's in the same directory as `config.json`
4. Execute `reverse_proxy`. You may want to add it as a service to run on boot.
5. If no errors appear, you're done! Now to setup the back end servers.

## Setting up the back end server
The back end server is what the actual Minecraft server is running on. You need to repeat these steps for every Minecraft server you wish to setup.

1. Create a new droplet called `{name}-automated` where you replace `{name}` with the name you chose in your front end's configuration file. For example if I had `"name": "vanilla",` in my configuration file, the droplet would be called `vanilla-automated`.
2. Download your preffered Minecraft server software. Note that you may need to install Java to run it.
3. Set up the server as you would normally by configuring the server.properties. It is recommended that you match the max players in server.properites with the one in your front end configuration.
4. Some Minecraft servers such as Spigot have a connection limit per IP address restriction. Make sure you turn this off else people will have trouble connecting.
5. Try running the Minecraft server to check that it works.
7. Download the sample back end configuration from [here](https://github.com/1lann/dynamicserver/blob/master/backend/config_sample.json).
8. Rename it to `config.json`.
6. Download/upload the `backend` software onto the server and make sure it's in the same directory as `config.json`.
9. Fill in the configuration to fit your needs. Configuration documentation is available here.
10. Make sure that the back end helper runs on startup. I do this by adding `/path/to/backend >> /path/to/backend.log 2>&1 &` to `/etc/rc.local`, which also writes logs to `/path/to/backend.log`.
12. Make sure your manually started minecraft server is not running.
11. Run `backend`, which should also start your minecraft server, and see if it is recognised by the front end server by adding the server's hostname to your Minecraft server list.
12. If the server appears on the list, and has the same status message specified in server.properties, then it is working!
13. If the server does not appear on the list, or the status is "Powered off", "Unavailable", or is stuck on "Starting up...", there may be an issue with your configuration. See troubleshooting for more help.
14. Try connecting to the server and play on it for a bit, then leave it and wait for the auto shutdown duration specified in your front end server's configuration, and see if it automatically shuts down!
15. If everything appears to be working, congratulations! You've set up an automatically managed dynamically launching Minecraft server.

# Inquiries
Need help? Have any questions or queries? Want to give praise, criticism, or feedback? Feel free to email me at me@chuie.io with anything, or create a new GitHub issue.

# License
dynamicserver is licensed under the MIT license that can be found [here](https://github.com/1lann/dynamicserver/blob/master/LICENSE).
