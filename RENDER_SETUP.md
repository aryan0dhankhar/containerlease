# Deploying to Render.com - Complete Guide

## Step 1: Deploy Blueprint (Backend + Frontend)

1. Go to **https://dashboard.render.com**
2. Click **"New +"** → **"Blueprint"**
3. Connect your GitHub account and select `containerlease` repo
4. Click **"Deploy"**
5. Wait 5-10 minutes for deployment to complete

Your services will be available at:
- Backend: `https://containerlease-backend.onrender.com`
- Frontend: `https://containerlease-frontend.onrender.com`

## Step 2: Create PostgreSQL Database

1. In Render dashboard, click **"New +"** → **"PostgreSQL"**
2. Fill in:
   - **Name:** `containerlease-postgres`
   - **Database:** `containerlease`
   - **User:** `containerlease`
   - **Plan:** Free
3. Click **"Create Database"**
4. Wait for it to be created (2-3 minutes)
5. Copy the **Internal Database URL** (format: `postgres://...`)

## Step 3: Create Redis Cache

1. In Render dashboard, click **"New +"** → **"Redis"**
2. Fill in:
   - **Name:** `containerlease-redis`
   - **Plan:** Free
3. Click **"Create Redis"**
4. Wait for it to be created
5. Copy the **Internal Redis URL** (format: `redis://...`)

## Step 4: Update Backend Environment Variables

1. Go to your **containerlease-backend** service
2. Click **"Settings"** → **"Environment"**
3. Update these variables:
   - `DB_HOST`: Extract from PostgreSQL Internal URL (hostname part)
   - `DB_USER`: `containerlease`
   - `DB_PASSWORD`: From PostgreSQL Internal URL
   - `DB_NAME`: `containerlease`
   - `REDIS_URL`: Paste the Redis Internal URL from Step 3
4. Keep `JWT_SECRET` as is (auto-generated)
5. Click **"Save"**

**Example from PostgreSQL URL `postgres://user:pass@host.render.internal:5432/db`:**
- `DB_HOST`: `host.render.internal`
- `DB_PORT`: `5432`
- `DB_USER`: `user`
- `DB_PASSWORD`: `pass`
- `DB_NAME`: `db`

## Step 5: Test Your App

1. Open frontend URL: `https://containerlease-frontend.onrender.com`
2. Try creating an account (use valid UUID for tenantId)
3. Login with credentials

## Step 6: Run Database Migrations (If Needed)

If you need to manually run migrations:

1. Get PostgreSQL connection details from Render dashboard
2. Run migration script (Render will auto-run on backend startup)
3. Check backend logs for "migration applied" message

---

## Troubleshooting

### Services not deployed?
- Check "Events" tab in Render dashboard
- View logs for error messages
- Ensure `render.yaml` is in repo root

### Backend can't connect to database?
- Verify `DB_HOST` uses `.render.internal` (internal hostname)
- Check credentials in environment variables
- Ensure database is in "Available" state

### Frontend can't reach backend?
- Check `VITE_BACKEND_URL` is correct
- Ensure backend service is "Live"
- Check browser console for exact error

### First request is slow?
- Free tier services spin down after 15 mins of inactivity
- First request wakes them up (takes 30 seconds)
- This is normal on free tier

---

## Next Steps

**To upgrade for continuous uptime:**
1. Select any service
2. Click "Settings"
3. Change Plan to "Starter" ($7+/month)
4. Service will run continuously without spin-down

**To add custom domain:**
1. Select service
2. Click "Settings" → "Custom Domain"
3. Add your domain
4. Update DNS records (Render will guide you)

---

Enjoy your deployed app! 🚀
