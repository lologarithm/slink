Slink game
--------------
Like slither.io -- written for fun beacuse I think I can do it. 

Implemented as a standalone app using unity3d instead of a web app.

Server
-------------
Go server, UDP network

Network Listener reads off network and passes messages around
Server Manager that creates new games, connects users to games.

Wrote a network protocol language and generator for fun (much like google protobuf). 
No reason to write my own other than to see if I can.

Flow
-------------
Users join a game, and collect food.
Collisions with other snakes will kill you (and you become food)


Features TODO List
--------------------------
1. Add minimap
