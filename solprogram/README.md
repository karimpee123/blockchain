<b>Solana Program Development Guide</b>
---
Complete step-by-step guide untuk setup dan develop Solana program.

📋 <b>Table of Contents</b>
---
1. Prerequisites
2. Installation
3. Project Setup
4. Development
5. Testing
6. Deployment
7. Troubleshooting
---
<b>Prerequisites</b>
---
- macOS, Linux, or WSL2 (Windows)
- Basic knowledge of Rust and TypeScript
- Terminal/Command line familiarity
---
<b>Installation</b>
---
1. Install Rust
```shell
# Install Rust via rustup
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# Restart terminal or run
source $HOME/.cargo/env

# Verify installation
rustc --version
cargo --version
```
2. Install Solana CLI
```shell
# Install Solana CLI tools
sh -c "$(curl -sSfL https://release.solana.com/stable/install)"

# Add to PATH (add to ~/.bashrc or ~/.zshrc)
export PATH="$HOME/.local/share/solana/install/active_release/bin:$PATH"

# Restart terminal and verify
solana --version
```
3. Configure Solana CLI
```shell
# Set to devnet (for development)
solana config set --url devnet

# Generate a new keypair
solana-keygen new

# Check your address
solana address

# Check balance
solana balance

# Airdrop SOL for testing (devnet only)
solana airdrop 2
```
4. Install Anchor Framework
```shell
# Install dependencies (macOS)
brew install node
brew install yarn

# Install Anchor Version Manager (avm)
cargo install --git https://github.com/coral-xyz/anchor avm --locked --force

# Install latest Anchor version
avm install latest
avm use latest

# Verify installation
anchor --version
```
5. Install Additional Tools
```shell
# TypeScript (for tests)
npm install -g typescript ts-node

# Prettier (code formatting)
npm install -g prettier
```
---
<b>Project Setup</b>
---
1. Create New Anchor Project
```shell
# Create new project
anchor init my_solana_project
cd my_solana_project

# Project structure:
# ├── Anchor.toml          # Anchor configuration
# ├── Cargo.toml           # Rust workspace config
# ├── programs/            # Your Solana programs
# │   └── my_solana_project/
# │       ├── Cargo.toml
# │       └── src/
# │           └── lib.rs   # Main program code
# ├── tests/               # TypeScript tests
# │   └── my_solana_project.ts
# └── target/              # Build output
```
2. Configure Anchor.toml
```javascript
[features]
seeds = false
skip-lint = false

[programs.localnet]
my_solana_project = "YourProgramIDHere"

[programs.devnet]
my_solana_project = "YourProgramIDHere"

[registry]
url = "https://api.apr.dev"

[provider]
cluster = "Devnet"
wallet = "~/.config/solana/id.json"

[scripts]
test = "yarn run ts-mocha -p ./tsconfig.json -t 1000000 tests/**/*.ts"
```
3. Update Program ID
```shell
# Build project first
anchor build

# Get your program ID
solana address -k target/deploy/my_solana_project-keypair.json

# Update in two places:
# 1. programs/my_solana_project/src/lib.rs
declare_id!("YOUR_PROGRAM_ID_HERE");

# 2. Anchor.toml
[programs.devnet]
my_solana_project = "YOUR_PROGRAM_ID_HERE"

# Build again after updating
anchor build
```
---
<b>Development</b>
---
1. Write Your Program (lib.rs)
```javascript
use anchor_lang::prelude::*;

declare_id!("YOUR_PROGRAM_ID");

#[program]
pub mod my_solana_project {
    use super::*;

    pub fn initialize(ctx: Context<Initialize>) -> Result<()> {
        msg!("Hello, Solana!");
        Ok(())
    }
}

#[derive(Accounts)]
pub struct Initialize {}
```
2. Build Program
```shell
# Clean previous builds
anchor clean

# Build program
anchor build

# Build with verbose output (for debugging)
anchor build --verbose
```
3. Common Commands
```shell
# Format code
cargo fmt

# Run clippy (linter)
cargo clippy

# Check for errors without building
cargo check

# Watch for changes and rebuild
cargo watch -x build
```
---
<b>Testing</b>
---
1. Local Testing
```shell
# Start local validator
solana-test-validator

# In another terminal, run tests
anchor test --skip-local-validator

# Or run all (will start validator automatically)
anchor test
```
2. Write Tests (TypeScript)
```javascript
import * as anchor from "@coral-xyz/anchor";
import { Program } from "@coral-xyz/anchor";
import { MyProject } from "../target/types/my_project";
import { expect } from "chai";

describe("my_project", () => {
  const provider = anchor.AnchorProvider.env();
  anchor.setProvider(provider);
  
  const program = anchor.workspace.MyProject as Program<MyProject>;

  it("Initialize test", async () => {
    const tx = await program.methods.initialize().rpc();
    console.log("Transaction signature:", tx);
  });
});
```
3. Test on Devnet
```shell
# Make sure connected to devnet
solana config set --url devnet

# Ensure you have SOL
solana balance
solana airdrop 2

# Run tests against devnet
anchor test --skip-local-validator
```
---
<b>Deployment</b>
---
1. Deploy to Devnet
```shell
# Set cluster to devnet
solana config set --url devnet

# Check balance (need ~2-5 SOL for deployment)
solana balance

# If needed, airdrop
solana airdrop 2

# Deploy program
anchor deploy

# Verify deployment
solana program show <PROGRAM_ID>
```
2. Deploy to Mainnet
```shell
# ⚠️ MAINNET - Real SOL required!

# Set cluster to mainnet
solana config set --url mainnet-beta

# Check balance (need sufficient SOL)
solana balance

# Deploy
anchor deploy

# Verify
solana program show <PROGRAM_ID>
```
3. Update Existing Program
```shell
# Make changes to your code
# Build
anchor build

# Deploy (will upgrade existing program)
anchor deploy

# Or use upgrade command
anchor upgrade target/deploy/my_program.so --program-id <PROGRAM_ID>
```
---
<b>Program Management</b>
---
1. Check Program Info
```shell
# Show program details
solana program show <PROGRAM_ID>

# Check program account balance
solana balance <PROGRAM_ID>

# Get program data size
solana program dump <PROGRAM_ID> program.so
ls -lh program.so
```
2. Close Program (Devnet/Localnet only)
```shell
# Recover SOL from closed program
solana program close <PROGRAM_ID> --bypass-warning

# This will return SOL to your wallet
```
3. Set Program Authority
```shell
# Transfer upgrade authority
solana program set-upgrade-authority <PROGRAM_ID> --new-upgrade-authority <NEW_AUTHORITY>

# Make program immutable (cannot be upgraded)
solana program set-upgrade-authority <PROGRAM_ID> --final
```
---
<b>Troubleshooting</b>
---
<b>Common Issues</b>
1. "Insufficient funds for rent"
```shell
solana airdrop 5
```
2. "Program already deployed"
```shell
# Option A: Close old program
solana program close <PROGRAM_ID> --bypass-warning

# Option B: Generate new keypair
solana-keygen new -o target/deploy/program-keypair.json --force
# Then update program ID in lib.rs and Anchor.toml
```
3. "Anchor version mismatch"
```shell
# Check versions
anchor --version
cargo tree | grep anchor

# Update Anchor
avm install latest
avm use latest

# Update Cargo.toml dependencies
[dependencies]
anchor-lang = "0.29.0"
```
4. "Transaction too large"
- Reduce account size
- Split into multiple instructions
- Optimize data structures
5. "Custom program error: 0x1"
- Check error messages in logs
- Use msg!() for debugging
- Review account constraints
---
<b style="font-size:1.2em">Debug Tips</b>
---
```shell
# View detailed logs
solana logs --url devnet

# View logs for specific program
solana logs <PROGRAM_ID> --url devnet

# Increase compute units in test
tx.add(
  ComputeBudgetProgram.setComputeUnitLimit({
    units: 1_400_000,
  })
);
```
Useful Commands Cheatsheet

```shell
# Solana CLI
solana config get                    # Show current config
solana config set --url <CLUSTER>    # Change cluster
solana balance                       # Check balance
solana airdrop <AMOUNT>             # Request airdrop
solana address                       # Show public key
solana-keygen new                    # Generate new keypair

# Anchor
anchor init <PROJECT>                # New project
anchor build                         # Build program
anchor test                          # Run tests
anchor deploy                        # Deploy to configured cluster
anchor clean                         # Clean build artifacts
anchor upgrade <PROGRAM_PATH>        # Upgrade program

# Cargo
cargo build-sbf                      # Build Solana program
cargo test                           # Run Rust tests
cargo fmt                            # Format code
cargo clippy                         # Run linter
```
---
<b>Resources</b>
---
- Solana Documentation
- Anchor Documentation
- Solana Cookbook
- Solana Program Library
- Anchor Examples
---
Project Example: Red Envelope Program
See <code>lib.rs</code> for a complete implementation of a multi-type envelope system with:
- Direct fixed transfers
- Group fixed distributions
- Group random distributions
- Claim and refund mechanisms