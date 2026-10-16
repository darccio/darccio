<!--
title: Buscar
description: "Busca en los títulos y resúmenes de los artículos, páginas y documentación de dario.cat."
language: es
nav: search
feed: false
atproto: false
-->

# Buscar

<form class="search-form" role="search" method="get" action="/es/search/">
<label for="search-query">¿Qué quieres encontrar?</label>
<div class="search-form__controls"><input id="search-query" type="search" name="q" value="" data-search-query maxlength="160" aria-describedby="search-hint"><button type="submit">Buscar</button></div>
</form>

<p id="search-hint">Busca en títulos y resúmenes en catalán, castellano e inglés. Por ejemplo: Mergo, Synology o política.</p>

<!-- SITE_SEARCH_STATUS_START --><p class="search-status">También puedes explorar todas las páginas del catálogo.</p><!-- SITE_SEARCH_STATUS_END -->

<!-- SITE_SEARCH_RESULTS_START -->
<ul class="search-results">
<embed src="../../.zas/partials/search-items.html" type="text/html">
</ul>
<!-- SITE_SEARCH_RESULTS_END -->
