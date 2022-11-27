export interface Env {
}

const payload: string = `{
  "subject": "acct:dario@mastodont.cat",
  "aliases": [
    "https://mastodont.cat/@dario",
    "https://mastodont.cat/users/dario"
  ],
  "links": [
    {
      "rel": "http://webfinger.net/rel/profile-page",
      "type": "text/html",
      "href": "https://mastodont.cat/@dario"
    },
    {
      "rel": "self",
      "type": "application/activity+json",
      "href": "https://mastodont.cat/users/dario"
    },
    {
      "rel": "http://ostatus.org/schema/1.0/subscribe",
      "template": "https://mastodont.cat/authorize_interaction?uri={uri}"
    }
  ]
}`

export const worker = {
	async fetch(
		request: Request,
		env: Env,
		ctx: ExecutionContext
	): Promise<Response> {
        let url : URL = new URL(request.url);
        if (url.searchParams.get('resource') !== 'acct:d@rio.hn') {
            return new Response('Not found', { status: 404 })
        }

		return new Response(payload, {
            headers: {
                'Content-Type': 'application/jrd+json; charset=utf-8'
            }
        });
	},
};

export default worker;
