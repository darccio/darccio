<!--
entry: true
description: "Com està fet dario.cat: Zas, Cloudflare Pages, codi i llicències."
updated: 2026-10-12T08:09:18+02:00
-->

<h1 class="p-name">Sobre aquest web</h1>

dario.cat és un web estàtic fet amb [Zas](https://github.com/darccio/zas), el meu generador escrit en Go. Les pàgines parteixen de Markdown o HTML i comparteixen una plantilla. El [codi del web](https://github.com/darccio/darccio) és públic.

## Publicació

Cloudflare Pages construeix el web quan es publiquen canvis a la branca `main`. La construcció executa `make -f .zas/Makefile publish` i publica `.zas/deploy`. La versió de Zas queda fixada al Makefile perquè cada construcció faci servir una revisió concreta; aquest web fa servir [Zas v0.0.9](https://github.com/darccio/zas/releases/tag/v0.0.9).

Les [guies dels projectes](/projects/) viuen als seus repositoris. El web només incorpora els fitxers seleccionats d'una revisió fixada i els renderitza com a dades, sense executar plantilles, connectors ni scripts del projecte. Cada guia conserva la llicència original i enllaça la revisió d'origen. Les actualitzacions arriben en una proposta de canvi al web; es publiquen quan es fusiona a `main`.

Abans de publicar, la construcció comprova els camins d'importació de Go que serveix aquest domini, com ara `dario.cat/mergo` i `dario.cat/darna`.

## Disseny i contingut

El web fa servir HTML i CSS, sense JavaScript propi al navegador. La navegació, el correu de contacte i les pàgines es poden llegir sense scripts. La fotografia està allotjada aquí mateix.

Escric el vessant personal en català, amb articles en castellà i anglès. La documentació dels projectes de Go és principalment en anglès.

Les [subscripcions](/feeds/) permeten seguir el contingut amb un lector de feeds. La [política d'IA](/ai/) explica les preferències per als rastrejadors i les dades personals.

Els articles tenen metadades compartides entre els feeds RSS, microformats i dades estructurades. Les versions Markdown i els fitxers de context per a agents es generen durant la construcció, sense scripts al navegador.

La publicació d'articles a Standard.site té un procés separat, posterior al desplegament i limitat als articles revisats. Una incidència al servei de publicació no impedeix publicar el web. Les còpies distribuïdes per l'AT Protocol queden fora de l'abast de les preferències dels rastrejadors d'aquest domini.

## Llicències

El contingut original sobre el qual tinc els drets necessaris es publica sota [CC BY-SA 4.0](https://creativecommons.org/licenses/by-sa/4.0/deed.ca). El codi original del web manté [l'Unlicense](/LICENSE-CODE). El CV, Lýd i el material de tercers tenen excepcions i condicions pròpies; [LICENSE](/LICENSE) en detalla l'abast.
