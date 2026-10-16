<!--
title: Search
description: "Search the titles and summaries of articles, pages and documentation on dario.cat."
language: en
nav: search
feed: false
atproto: false
-->

# Search

<form class="search-form" role="search" method="get" action="/en/search/" toolname="search_site" tooldescription="Search public article, page and documentation titles and summaries on dario.cat in Catalan, Spanish and English." toolautosubmit>
<label for="search-query">What are you looking for?</label>
<div class="search-form__controls"><input id="search-query" type="search" name="q" value="" data-search-query maxlength="160" aria-describedby="search-hint" toolparamdescription="Words to find in titles and summaries; up to 160 characters."><button type="submit">Search</button></div>
</form>

<p id="search-hint">Search titles and summaries in Catalan, Spanish and English. For example: Mergo, Synology or politics.</p>

<!-- SITE_SEARCH_STATUS_START --><p class="search-status">You can also browse every page in the catalogue.</p><!-- SITE_SEARCH_STATUS_END -->

<!-- SITE_SEARCH_RESULTS_START -->
<ul class="search-results">
<embed src="../../.zas/partials/search-items.html" type="text/html">
</ul>
<!-- SITE_SEARCH_RESULTS_END -->
