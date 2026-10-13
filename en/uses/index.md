<!--
entry: true
nav: ""
language: en
description: "Tools documented on dario.cat: the website built with Zas and Synology NAS monitoring with Datadog."
updated: 2026-10-11T22:53:31+02:00
-->

<h1 class="p-name">Tools</h1>

A collection of tools I have written about on this website.

## This website

This website is built with [Zas](https://github.com/darccio/zas), my static site generator written in Go. I write the pages in Markdown or HTML, keep the [code on GitHub](https://github.com/darccio/darccio), and publish it with Cloudflare Pages.

The [colophon](/en/colophon/) explains how the website is built and published.

## NAS monitoring

In the article from 16 August 2026, [Monitoring a Synology NAS with Datadog](/en/posts/monitoring-synology-nas-with-datadog/), I explain how I configured my Synology NAS:

- DSM and Container Manager to run containers.
- The Datadog Agent in a container to monitor processes and services.
- SNMPv3 to collect disk, RAID and temperature metrics.

The article includes the configuration, checks and problems I encountered.
