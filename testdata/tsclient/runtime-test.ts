import { UsersClient } from './users'

async function main() {
  const calls = []
  const fetchImpl = (async (url, init) => {
    calls.push({ url: String(url), init })
    return { ok: true, status: 200, json: async () => ({ id: 1 }) }
  })
  const client = new UsersClient({ baseUrl: 'http://example.com', fetch: fetchImpl })
  await client.get_users_id({ id: 123 }, { authorization: 'token' }, { q: 'hi' })
  if (calls.length !== 1) {
    throw new Error('expected one request')
  }
  const call = calls[0]
  if (call.url !== 'http://example.com/users/123?q=hi') {
    throw new Error('unexpected url: ' + call.url)
  }
  if (!call.init || !call.init.headers || call.init.headers.authorization !== 'token') {
    throw new Error('missing auth header')
  }
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
