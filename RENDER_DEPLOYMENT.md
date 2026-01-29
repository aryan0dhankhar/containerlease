# Render.com Deployment Guide

## ✨ Free Deployment in 5 Minutes

This guide walks you through deploying ContainerLease to Render.com for free.

## Prerequisites

- GitHub account (free)
- Render account (free at https://render.com)
- Code pushed to GitHub

## Step-by-Step Deployment

### 1. Create GitHub Repository

```bash
# Initialize git in your project
cd /Users/aryandhankhar/Documents/dev/containerlease
git init
git add .
git commit -m "Initial commit: ContainerLease project"

# Create repository on GitHub
# Go to https://github.com/new
# Name: containerlease
# Click "Create repository"

# Push code to GitHub
git remote add origin https://github.com/YOUR_USERNAME/containerlease.git
git branch -M main
git push -u origin main
```

### 2. Sign Up for Render

1. Go to https://render.com
2. Click "Sign Up" → Choose "Sign up with GitHub"
3. Authorize Render to access your GitHub account
4. Create account

### 3. Deploy from `render.yaml`

1. In Render dashboard, click **"New +"** → **"Blueprint"**
2. Select your `containerlease` repository
3. Click "Connect"
4. Choose branch: `main`
5. Click "Deploy"

**That's it!** Render will:
- ✅ Create PostgreSQL database
- ✅ Create Redis cache
- ✅ Build and deploy backend
- ✅ Build and deploy frontend
- ✅ Set up environment variables automatically
- ✅ Assign free subdomains

### 4. Get Your URLs

Once deployment completes (5-10 minutes):
- **Frontend:** https://containerlease-frontend.onrender.com
- **Backend API:** https://containerlease-backend.onrender.com
- **Database:** Automatically managed

### 5. Test Your Application

```bash
# Test backend health
curl https://containerlease-backend.onrender.com/healthz

# Test frontend (open in browser)
https://containerlease-frontend.onrender.com
```

## Auto-Deploy on Code Push

The `render.yaml` file enables automatic redeployment whenever you push to GitHub:

```bash
# Make changes
git add .
git commit -m "Feature: add new feature"
git push origin main

# Render automatically redeploys! 🚀
```

## Scaling Up (When Ready)

When you move beyond free tier:

1. Go to service in Render dashboard
2. Click "Settings"
3. Change plan from "Free" to "Starter" ($7+/month)
4. Services stay running instead of spinning down

## Monitoring & Logs

1. In Render dashboard, click your service
2. Click "Logs" tab to see real-time logs
3. Click "Metrics" to monitor CPU/Memory

## Custom Domain (Optional)

1. Register domain (Namecheap, Google Domains, etc.)
2. In Render service, click "Settings"
3. Add custom domain
4. Update DNS records (Render will guide you)

## Troubleshooting

### Services Won't Deploy
- Check "Events" tab for error messages
- Ensure `render.yaml` is in project root
- Verify GitHub permissions

### Frontend Can't Reach Backend
- Check that backend URL is correctly set in environment
- Verify both services are deployed
- Check CORS settings in backend

### Database Connection Issues
- Render provides connection string automatically
- Check environment variable names match exactly
- Wait 2-3 minutes for PostgreSQL to initialize

### Redeploy Manually
1. Go to service in Render dashboard
2. Click "Manual Deploy" → "Deploy latest"

## Important Notes

⚠️ **Free Tier Limitations:**
- Services spin down after 15 minutes of inactivity
- First request after spin-down takes 30 seconds
- Suitable for learning/development, not production

📈 **When you're ready for production:**
1. Upgrade to "Starter" plan ($7-12/month per service)
2. Services run continuously
3. Better performance & reliability

## Need Help?

- Render Docs: https://render.com/docs
- GitHub Issues: Create issue in repository
- Discord Community: Render's support channel

---

**Your app is now live on the internet!** 🎉
