# dynamicserver
A system to dynamically launch DigitalOcean droplets as Minecraft servers if people are trying to connect, and destroys them when there's nobody on the server to save running costs.

Due to the nature of this application, I have also made my own implementation of the Minecraft network protocol including pinging, and logging in without authentication, for protocol version 5. The current state of the code for this is currently garbage as I just wanted to make a quick proof of concept, but hopefully I will fix it soon to make it passable. But hey, it works!
