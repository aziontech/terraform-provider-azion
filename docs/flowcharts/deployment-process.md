# Terraform Provider Azion - Deployment Process

This flowchart shows the two deployment paths:
1. **Automatic** - Merge to `main` → auto version bump → release
2. **Manual** - Push tag or workflow dispatch → release

```mermaid
flowchart TD
    Start([Developer Action]) --> Decision{Action Type?}
    
    %% Pull Request Path
    Decision -->|Create PR| PR[Pull Request]
    PR --> Checks[Code Checks & Tests]
    Checks -->|Pass| ReadyMerge[Ready to Merge]
    Checks -->|Fail| PR
    
    %% Automatic Deployment - Merge to Main
    Decision -->|Merge to main| MainBranch[Merge to main]
    ReadyMerge --> MainBranch
    
    MainBranch --> DeployMain[Deploy to Production Workflow]
    DeployMain --> GoReport[Go Report Card]
    GoReport --> BumpVersion[Auto Bump Version]
    BumpVersion --> AutoTag[Create Tag v*]
    
    %% Manual Release Path
    Decision -->|Push Tag v*| ManualTag[Manual Tag]
    Decision -->|Workflow Dispatch| ManualDispatch[Manual Trigger]
    
    %% Both paths converge
    AutoTag --> Release[Release Workflow]
    ManualTag --> Release
    ManualDispatch --> Release
    
    %% Release Process
    Release --> GoReleaser[GoReleaser]
    GoReleaser --> Sign[Sign with GPG]
    Sign --> Publish[Publish to GitHub/Terraform Registry]
    Publish --> End([Complete])
    
    %% Styling
    classDef autoStyle fill:#d4edda,stroke:#28a745,stroke-width:2px
    classDef manualStyle fill:#fff3cd,stroke:#ffc107,stroke-width:2px
    classDef prStyle fill:#e1f5ff,stroke:#0366d6,stroke-width:2px
    
    class MainBranch,DeployMain,GoReport,BumpVersion,AutoTag autoStyle
    class ManualTag,ManualDispatch manualStyle
    class PR,Checks,ReadyMerge prStyle
```

## Key Differences

| Path | Trigger | Version Bump | Use Case |
|------|---------|--------------|----------|
| **Automatic** | Merge to `main` | Automatic | Continuous deployment |
| **Manual** | Tag push or dispatch | Manual | Controlled releases |

## Workflows

- **PR Validation**: Code checks, tests, linting
- **Deploy Main**: Auto version bump on merge
- **Release**: GoReleaser with GPG signing
