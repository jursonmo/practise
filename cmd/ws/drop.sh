#! /bin/sh
iptables -A INPUT -p tcp  --dport 8080 -j DROP
iptables -A INPUT -p tcp  --sport 8080 -j DROP