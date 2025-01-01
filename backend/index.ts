import { Elysia } from 'elysia'
import { z } from 'zod'
import jwt from 'jsonwebtoken'
import fs from 'fs'
import path from 'path'

const STEAM_API_KEY = process.env.STEAM_API_KEY
if(!STEAM_API_KEY) {
  console.error('Missing STEAM_API_KEY environment variable')
  process.exit(1)
}
const JWT_SECRET = process.env.JWT_SECRET
if(!JWT_SECRET) {
  console.error('Missing JWT_SECRET environment variable')
  process.exit(1)
}

const app = new Elysia()
  .onError(({ error, code, set }) => {
    if(code === 'NOT_FOUND') {
      set.status = 404
      return { ok: false, error: 'Not found' }
    } else {
      set.status = 500
      console.error(error)
      return { ok: false, error: 'Internal server error' }
    }
  })
const host = process.env.PUBLIC_URL || 'http://localhost:3001'
const redirectURL = '/auth/steam/callback'

app.get('/auth/steam', async ({ redirect, query }) => {
  let returnTo = host + redirectURL
  if(query.file) {
    returnTo += '?file=' + query.file
  }
  const redirectUrl = `https://steamcommunity.com/openid/login?openid.ns=http://specs.openid.net/auth/2.0&openid.mode=checkid_setup&openid.return_to=${encodeURIComponent(returnTo)}&openid.realm=${host}&openid.identity=http://specs.openid.net/auth/2.0/identifier_select&openid.claimed_id=http://specs.openid.net/auth/2.0/identifier_select`
  return redirect(redirectUrl)
})

const cache = new Map<string, { expires: number, owns: boolean }>()
async function checkIfOwnsAdon(steamId: string): Promise<boolean | 'unknown'> {
  const cachedResponse = cache.get(steamId)
  if (cachedResponse) {
    if (cachedResponse.expires > Date.now()) {
      return cachedResponse.owns
    }
  }
  const response = await fetch('https://api.steampowered.com/IPlayerService/GetOwnedGames/v1/?' + new URLSearchParams({
    key: STEAM_API_KEY as string,
    input_json: JSON.stringify({ 
      steamid: steamId,
      appids_filter: [1450830],
      include_appinfo: false,
      include_played_free_games: false
    }),
  }))
    .then((res) => res.json() as Promise<{ response: { games: { appid: number }[] } }>)

  if(!('games' in response.response) || !Array.isArray(response.response.games)) {
    return 'unknown'
  }

  const ownsAdon = response.response.games.some(game => game.appid === 1450830)
  cache.set(steamId, { expires: Date.now() + 1000 * 60 * 60 * 24 * 7, owns: ownsAdon })

  return ownsAdon
}

app.get('/auth/steam/callback', async ({ redirect, set, query, cookie }) => {
  try {
    const reqQuery = await z.object({
      'openid.ns': z.string(),
      'openid.mode': z.string(),
      'openid.op_endpoint': z.string(),
      'openid.claimed_id': z.string(),
      'openid.identity': z.string(),
      'openid.return_to': z.string(),
      'openid.response_nonce': z.string(),
      'openid.assoc_handle': z.string(),
      'openid.signed': z.string(),
      'openid.sig': z.string(),
      'file': z.string().optional()
    }).safeParseAsync(query)
    if (!reqQuery.success) {
      set.status = 400
      return { ok: false }
    }

    const validateParams = new URLSearchParams()
    validateParams.append('openid.ns', reqQuery.data['openid.ns'])
    validateParams.append('openid.mode', 'check_authentication')
    validateParams.append('openid.op_endpoint', reqQuery.data['openid.op_endpoint'])
    validateParams.append('openid.claimed_id', reqQuery.data['openid.claimed_id'])
    validateParams.append('openid.identity', reqQuery.data['openid.identity'])
    validateParams.append('openid.return_to', reqQuery.data['openid.return_to'])
    validateParams.append('openid.response_nonce', reqQuery.data['openid.response_nonce'])
    validateParams.append('openid.assoc_handle', reqQuery.data['openid.assoc_handle'])
    validateParams.append('openid.signed', reqQuery.data['openid.signed'])
    validateParams.append('openid.sig', reqQuery.data['openid.sig'])

    const signedFields = reqQuery.data['openid.signed'].split(',')
    for (const field of signedFields) {
      const key = `openid.${field}`
      if (key in reqQuery.data) {
        const value = reqQuery.data[key as keyof typeof reqQuery.data] as string
        validateParams.append(`openid.${field}`, value)
      }
    }

    const verificationRequest = await fetch('https://steamcommunity.com/openid/login', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
      },
      body: validateParams.toString(),
    })
    const verificationResponse = await verificationRequest.text()
    const isValid = verificationResponse.includes('is_valid:true')
    if(!isValid) {
      set.status = 401
      return { ok: false, error: 'Invalid state. Either you pressed "back" button in your browser or Steam broke OpenID authorization and our server can\'t verify your steam authorization token. Please return to the main page and try again or try later. Other browsers might help too, but this is a Steam problem.', errorRu: 'Ошибка. Или вы нажали "назад" в браузере, или у Steam сломались серверы авторизации OpenID и наш сервер не может подтвердить ваш токен. Пожалуйста, вернитесь на главную и попробуйте еще раз или попробуйте позже. Использование других браузеров также может помочь, но это, вероятнее всего, проблема Steam.' }
    }

    const steamId = reqQuery.data['openid.claimed_id'].match(/\/(\d+)$/)?.[1]
    if (!steamId) {
      set.status = 400
      return { ok: false }
    }

    const ownsAdon = await checkIfOwnsAdon(steamId)

    if (ownsAdon === true) {
      const token = jwt.sign({ steamId }, JWT_SECRET, { expiresIn: 365 * 24 * 60 * 60 })
      cookie['adon-mods-token'].set({
        value: token,
        httpOnly: true,
        maxAge: 365 * 24 * 60 * 60
      })
      if (reqQuery.data.file) {
        return redirect(host + '/download-page?file=' + reqQuery.data.file)
      } else {
        return redirect(host)
      }
    } else if (ownsAdon === 'unknown') {
      return redirect(host + '/purchase?purchased=unknown')
    } else {
      return redirect(host + '/purchase?purchased=false')
    }
  } catch (error) {
    console.error(error)
    set.status = 500
    return { ok: false, error: 'Internal server error' }
  }
})

const protectedDownloads = {
  'At Dead Of Night.app': './At Dead Of Night.zip',
  'atdeadofdeeznuts.7z': './atdeadofdeeznuts.7z',
  'at-dead-of-night-android-v1.0.0-t.me_adonru.apk': './at-dead-of-night-android-v1.0.0-t.me_adonru.apk',
  'at-dead-of-night-ios-v1.0.0-t.me_adonru.ipa': './at-dead-of-night-ios-v1.0.0-t.me_adonru.ipa',
}

app.get('/download', async ({ cookie, redirect, query, set, headers }) => {
  const token = cookie['adon-mods-token'].value
  if (!token) {
    return redirect(host + '/purchase?file=' + query.file)
  }
  
  try {
    let decoded = await jwt.verify(token, JWT_SECRET)
    if (typeof decoded === 'string') decoded = JSON.parse(decoded)
    const tokenData = await z.object({
      steamId: z.string().regex(/^\d+$/)
    }).safeParseAsync(decoded)
    if (!tokenData.success) {
      throw new Error('Invalid token data')
    }
    if (await checkIfOwnsAdon(tokenData.data.steamId) === false) {
      cookie['adon-mods-token'].remove()
      return redirect(host + '/purchase?file=' + query.file)
    } else {
      jwt.sign({ steamId: tokenData.data.steamId }, JWT_SECRET, { expiresIn: 365 * 24 * 60 * 60 })
      cookie['adon-mods-token'].set({
        value: token,
        httpOnly: true,
        maxAge: 365 * 24 * 60 * 60
      })
    }
  } catch(e) {
    set.status = 400
    return { ok: false, error: 'Invalid cookie. Please clear the website\'s cookies and sign in with Steam again.' }
  }


  const fileNameParsing = await z.string().safeParseAsync(query.file)
  if (!fileNameParsing.success || !Object.hasOwn(protectedDownloads, fileNameParsing.data)) {
    set.status = 404
    return { ok: false, error: 'File not found' }
  }

  const filePath = path.resolve(__dirname, '../static', protectedDownloads[fileNameParsing.data as keyof typeof protectedDownloads])
  const fileStat = await fs.promises.stat(filePath)

  const range = headers['range'] 
  let start = 0
  let end = fileStat.size - 1

  if (range) {
    const rangeParts = range.replace(/bytes=/, '').split('-')
    start = parseInt(rangeParts[0], 10)
    end = rangeParts[1] ? parseInt(rangeParts[1], 10) : end

    if (start >= fileStat.size || end >= fileStat.size) {
      set.headers['Content-Range'] = `bytes */${fileStat.size}`
      set.status = 416
      return { ok: false, error: 'Requested Range Not Satisfiable' }
    }

    set.headers['Content-Range'] = `bytes ${start}-${end}/${fileStat.size}`
    set.headers['Content-Length'] = String(end - start + 1)
  } else {
    set.headers['Content-Length'] = String(fileStat.size)
  }

  // set.headers['Accept-Ranges'] = 'bytes'
  set.headers['Content-Type'] = 'application/octet-stream'
  set.headers['Content-Disposition'] = `attachment; filename="${path.basename(filePath)}"`

  let resultingFile = Bun.file(filePath)
  if (range) {
    resultingFile = resultingFile.slice(start, end + 1)
  }

  return resultingFile
})

app.listen(process.env.PORT || 3001)
console.log('Server running on http://localhost:' + (process.env.PORT || 3001))
