const PAGE_ROUTES = /* SITE_PAGE_ROUTES */ {};
export default {
  async fetch(request, env) {
    return env.ASSETS.fetch(request);
  },
};
