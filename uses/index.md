<!--
entry: true
description: "Eines documentades a dario.cat: el web amb Zas i la monitorització d'un NAS Synology amb Datadog."
updated: 2026-10-11T22:53:31+02:00
-->

<h1 class="p-name">Eines</h1>

Un recull de les eines que he explicat en aquest web.

## El web

Aquest web està fet amb [Zas](https://github.com/darccio/zas), el meu generador estàtic escrit en Go. Escric les pàgines en Markdown o HTML, mantinc el [codi a GitHub](https://github.com/darccio/darccio) i el publico amb Cloudflare Pages.

El [colofó](/colophon/) explica la construcció i la publicació del web.

## Monitorització del NAS

A l'article del 16 d'agost de 2026, [Monitoring a Synology NAS with Datadog](/en/posts/monitoring-synology-nas-with-datadog/), explico la configuració del meu NAS Synology:

- DSM i Container Manager per executar els contenidors.
- L'Agent de Datadog en un contenidor per observar processos i serveis.
- SNMPv3 per recollir les mètriques dels discos, RAID i temperatures.

L'article inclou la configuració, les comprovacions i els problemes que em vaig trobar.
