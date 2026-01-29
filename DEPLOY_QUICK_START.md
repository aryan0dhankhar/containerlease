# ContainerLease - Quick Start Deployment

## 🚀 Deploy in 5 Steps

### Step 1: Prepare Your Repository
```bash
cd /Users/aryandhankhar/Documents/dev/containerlease

# Check git status
git status

# Add all files
git add .

# Commit
git commit -m "ContainerLease - ready for deployment"
```

### Step 2: Push to GitHub
```bash
# If you haven't set remote yet:
git remote add origin https://github.com/YOUR_USERNAME/containerlease.git

# Push to GitHub
git push -u origin main
```

### Step 3: Create Render Account
- Visit https://render.com
- Sign up with GitHub
- Authorize access

### Step 4: Deploy Blueprint
1. In Render dashboard: **"New +"** → **"Blueprint"**
2. Select `containerlease` repository
3. Click **"Connect"**
4. Click **"Deploy"**

### Step 5: Wait & Access
- Wait 5-10 minutes for deployment
- Access frontend at: `https://containerlease-frontend.onrender.com`
- Access backend at: `https://containerlease-backend.onrender.com`

## ✅ What Gets Deployed

- **PostgreSQL Database** - Free, automatically managed
- **Redis Cache** - Free, automatically managed  
- **Backend API** (Go) - Free tier with auto spin-down
- **Frontend** (React) - Free tier with auto spin-down

## 📝 Key Files

- `render.yaml` - Deployment configuration (already created ✓)
- `.env.example` - Environment variable reference
- `RENDER_DEPLOYMENT.md` - Full deployment guide

## 🔄 Auto-Deploys

Push code to GitHub, Render automatically redeploys:
```bash
git add .
git commit -m "New feature"
git push origin main
# Automatic redeploy starts! 🚀
```

## 📊 Monitor Your App

1. Render Dashboard → Select service
2. View "Logs" → See real-time output
3. View "Metrics" → CPU, Memory usage
4. View "Events" → Deployment history

## ⚠️ Important

- **Free tier:** Services spin down after 15 min inactivity (first request takes 30s)
- **No custom domain** on free tier (use Render subdomain)
- **Suitable for:** Learning, development, portfolio
- **Perfect for:** Showing your project to others

## 🆘 Need Help?

See `RENDER_DEPLOYMENT.md` for detailed guide and troubleshooting

---

**Your app will be live within 10 minutes!** 🎉
