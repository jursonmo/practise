#! /bin/sh
iptables -D INPUT -p tcp  --dport 8080 -j DROP
iptables -D INPUT -p tcp  --sport 8080 -j DROP