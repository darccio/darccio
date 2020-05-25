sync:
	rsync -avzP --delete-during .zas/deploy/ root@rio.hn:/var/www/html/da.rio.hn/