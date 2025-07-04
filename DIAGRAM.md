## User Setup

```
User Installs DriftGuard
         ↓
Connect to Kubernetes
         ↓
Connect to Git Repository
         ↓
System Starts Monitoring
```

## Core Monitoring Loop

```
Monitor K8s Resources
         ↓
   Change Detected?
    ↙         ↘
  No           Yes
   ↓            ↓
Continue    Compare with Git
Monitoring       ↓
   ↑        Drift Found?
   ↑       ↙         ↘
   ↑     No           Yes
   ↑      ↓            ↓
   ↑ Log Normal    Analyze Risk Level
   ↑   State           ↓
   ↗      ↓            ↓
         ↗        Risk Level?
                 ↙    ↓    ↘
              Low   Medium  High
               ↓      ↓      ↓
         Dashboard Email/  Urgent
          Alert   Slack  Notification
               ↓      ↓      ↓
               ↘      ↓      ↙
                User Views Dashboard

```

## User Action Flow

```
User Views Dashboard
         ↓
    What to do?
   ↙   ↓   ↓   ↘
Investigate Auto Manual Ignore
   ↓      Fix   Fix    ↓
Show      ↓     ↓    Mark as
Detailed   ↓     ↓   Accepted
Diff      ↓     ↓      ↓
   ↓       ↓     ↓      ↓
Show Root  ↓     ↓   Continue
Cause     ↓     ↓   Monitoring
   ↓       ↓     ↓      ↓
Fix Decision?   ↓      ↓
↙    ↓    ↘     ↓      ↓
Auto Manual Ignore Update Log
↓     ↓     ↓    Git  Resolution
↓     ↓     ↓    Repo    ↓
↓     ↓     ↓     ↓      ↓
↓     ↓     ↓     ↓      ↓
Apply Git   Continue    ↓
Config      Monitoring  ↓
↓           ↓          ↓
Verify Fix  ↓          ↓
Success     ↓          ↓
↓           ↓          ↓
↘           ↓          ↙
  Log Resolution ←----↙
         ↓
   Back to Monitoring
```
