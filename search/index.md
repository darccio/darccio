<!--
title: Cerca
description: "Cerca als títols i resums dels articles, les pàgines i la documentació de dario.cat."
language: ca
nav: search
feed: false
atproto: false
-->

# Cerca

<form class="search-form" role="search" method="get" action="/search/">
<label for="search-query">Què vols trobar?</label>
<div class="search-form__controls"><input id="search-query" type="search" name="q" value="" data-search-query maxlength="160" aria-describedby="search-hint"><button type="submit">Cerca</button></div>
</form>

<p id="search-hint">Cerca als títols i resums en català, castellà i anglès. Per exemple: Mergo, Synology o política.</p>

<!-- SITE_SEARCH_STATUS_START --><p class="search-status">També pots explorar totes les pàgines del catàleg.</p><!-- SITE_SEARCH_STATUS_END -->

<!-- SITE_SEARCH_RESULTS_START -->
<ul class="search-results">
<embed src="../.zas/partials/search-items.html" type="text/html">
</ul>
<!-- SITE_SEARCH_RESULTS_END -->
