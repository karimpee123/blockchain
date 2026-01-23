use anchor_lang::prelude::*;
use anchor_lang::solana_program::system_instruction;

declare_id!("8sVfWmonJAzAQnS4nYcxv3GBSs4rDpvmniRrApwrh1QK");

pub const MAX_CREATE_AMOUNT: u64 = 10_000_000_000; // 10 SOL

#[program]
pub mod sols_multi_type {
    use super::*;

    pub fn init_user_state(ctx: Context<InitUserState>) -> Result<()> {
        let user_state = &mut ctx.accounts.user_state;
        user_state.owner = ctx.accounts.user.key();
        user_state.last_envelope_id = 0;
        Ok(())
    }

    pub fn create(
        ctx: Context<CreateEnvelope>,
        envelope_type: EnvelopeType,
        expiry_hours: u64,
    ) -> Result<()> {
        let user = &ctx.accounts.user;
        let user_state = &mut ctx.accounts.user_state;
        let envelope = &mut ctx.accounts.envelope;

        require_keys_eq!(user_state.owner, user.key(), CustomError::InvalidOwner);

        let envelope_id = user_state.last_envelope_id + 1;

        let total_amount: u64 = match &envelope_type {
            EnvelopeType::DirectFixed { amount, .. } => *amount,
            EnvelopeType::GroupFixed {
                total_users,
                amount_per_user,
            } => total_users
                .checked_mul(*amount_per_user)
                .ok_or(CustomError::MathOverflow)?,
            EnvelopeType::GroupRandom { total_amount, .. } => *total_amount,
        };

        require!(total_amount <= MAX_CREATE_AMOUNT, CustomError::ExceedMaxCreate);

        // TRANSFER SOL FROM USER TO ENVELOPE PDA
        let transfer_ix = system_instruction::transfer(
            &user.key(),
            &envelope.key(),
            total_amount,
        );

        anchor_lang::solana_program::program::invoke(
            &transfer_ix,
            &[
                user.to_account_info(),
                envelope.to_account_info(),
            ],
        )?;

        envelope.owner = user.key();
        envelope.envelope_id = envelope_id;
        envelope.envelope_type = envelope_type;
        envelope.amount = total_amount;
        envelope.total_claimed = 0;
        envelope.withdrawn_amount = 0;
        envelope.claimed_users = vec![];

        let clock = Clock::get()?;
        envelope.expiry = clock.unix_timestamp + (expiry_hours as i64 * 3600);

        user_state.last_envelope_id = envelope_id;

        msg!("Envelope created. Owner={}, ID={}, Amount={}", user.key(), envelope_id, total_amount);
        Ok(())
    }

    pub fn claim(ctx: Context<Claim>) -> Result<()> {
        let clock = Clock::get()?;
        let envelope = &mut ctx.accounts.envelope;
        let claimer = &ctx.accounts.claimer;

        // 1. CEK EXPIRY
        require!(clock.unix_timestamp < envelope.expiry, CustomError::Expired);

        // 2. CEK SUDAH CLAIM ATAU BELUM
        require!(
            !envelope.claimed_users.contains(&claimer.key()),
            CustomError::AlreadyClaimed
        );

        let claimed_len = envelope.claimed_users.len();

        // 3. VALIDASI BERDASARKAN ENVELOPE TYPE
        let claim_amount = match &envelope.envelope_type {
            EnvelopeType::DirectFixed { allowed_address, amount } => {
                require_keys_eq!(
                    *allowed_address,
                    claimer.key(),
                    CustomError::NotAllowed
                );
                *amount
            }

            EnvelopeType::GroupFixed {
                total_users,
                amount_per_user,
            } => {
                require!(
                    claimed_len < (*total_users as usize),
                    CustomError::QuotaFull
                );
                *amount_per_user
            }

            EnvelopeType::GroupRandom { total_users, .. } => {
                require!(
                    claimed_len < (*total_users as usize),
                    CustomError::QuotaFull
                );

                let remaining_users = (*total_users as usize) - claimed_len;
                let remaining_amount = envelope.amount - envelope.total_claimed;

                if remaining_users == 1 {
                    remaining_amount
                } else {
                    let max_per_user = remaining_amount / remaining_users as u64;
                    let rand_seed = (clock.unix_timestamp as u64)
                        .wrapping_mul(claimer.key().to_bytes()[0] as u64);
                    let rand_amount = (rand_seed % max_per_user) + 1;
                    rand_amount.min(remaining_amount)
                }
            }
        };

        // 4. CEK SUFFICIENT BALANCE
        require!(
            claim_amount <= (envelope.amount - envelope.total_claimed),
            CustomError::InsufficientFunds
        );

        // 5. TRANSFER SOL FROM ENVELOPE PDA TO CLAIMER
        **envelope.to_account_info().try_borrow_mut_lamports()? -= claim_amount;
        **claimer.to_account_info().try_borrow_mut_lamports()? += claim_amount;

        // 6. UPDATE STATE
        envelope.total_claimed += claim_amount;
        envelope.withdrawn_amount += claim_amount;
        envelope.claimed_users.push(claimer.key());

        msg!(
            "Claim success. Claimer={}, Amount={}, Type={:?}, Total claimed={}/{}",
            claimer.key(),
            claim_amount,
            envelope.envelope_type,
            envelope.total_claimed,
            envelope.amount
        );

        Ok(())
    }

    pub fn refund(ctx: Context<Refund>) -> Result<()> {
        let clock = Clock::get()?;
        let envelope = &mut ctx.accounts.envelope;
        let owner = &ctx.accounts.owner;

        // 1. CEK SUDAH EXPIRED
        require!(clock.unix_timestamp >= envelope.expiry, CustomError::NotExpired);

        // 2. CEK INI OWNER YANG BENAR
        require_keys_eq!(envelope.owner, owner.key(), CustomError::InvalidOwner);

        // 3. HITUNG SISA BALANCE
        let remaining = envelope.amount - envelope.total_claimed;
        require!(remaining > 0, CustomError::NothingToRefund);

        // 4. TRANSFER BALIK KE OWNER
        **envelope.to_account_info().try_borrow_mut_lamports()? -= remaining;
        **owner.to_account_info().try_borrow_mut_lamports()? += remaining;

        // 5. UPDATE STATE
        envelope.total_claimed = envelope.amount;

        msg!("Refund success. Owner={}, Amount={}", owner.key(), remaining);
        Ok(())
    }
}

// =========================
// ENUMS & STRUCTS
// =========================

#[derive(AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, Debug)]
pub enum EnvelopeType {
    DirectFixed {
        allowed_address: Pubkey,
        amount: u64,
    },
    GroupFixed {
        total_users: u64,
        amount_per_user: u64,
    },
    GroupRandom {
        total_users: u64,
        total_amount: u64,
    },
}

#[account]
pub struct UserState {
    pub owner: Pubkey,
    pub last_envelope_id: u64,
}

#[account]
pub struct EnvelopeAccount {
    pub owner: Pubkey,
    pub envelope_id: u64,
    pub envelope_type: EnvelopeType,
    pub amount: u64,
    pub withdrawn_amount: u64,
    pub total_claimed: u64,
    pub expiry: i64,
    pub claimed_users: Vec<Pubkey>,
}

// =========================
// ACCOUNT CONTEXTS
// =========================

#[derive(Accounts)]
pub struct InitUserState<'info> {
    #[account(
        init,
        payer = user,
        space = 8 + 32 + 8,
        seeds = [b"user_state", user.key().as_ref()],
        bump
    )]
    pub user_state: Account<'info, UserState>,

    #[account(mut)]
    pub user: Signer<'info>,

    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct CreateEnvelope<'info> {
    #[account(
        mut,
        seeds = [b"user_state", user.key().as_ref()],
        bump
    )]
    pub user_state: Account<'info, UserState>,

    #[account(
        init,
        payer = user,
        space = 8 + 32 + 8 + 50 + 8 + 8 + 8 + 8 + (4 + 32 * 10),
        seeds = [
            b"envelope",
            user.key().as_ref(),
            &(user_state.last_envelope_id + 1).to_le_bytes()
        ],
        bump
    )]
    pub envelope: Account<'info, EnvelopeAccount>,

    #[account(mut)]
    pub user: Signer<'info>,

    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct Claim<'info> {
    #[account(
        mut,
        seeds = [
            b"envelope",
            envelope.owner.as_ref(),
            &envelope.envelope_id.to_le_bytes()
        ],
        bump
    )]
    pub envelope: Account<'info, EnvelopeAccount>,

    #[account(mut)]
    pub claimer: Signer<'info>,
}

#[derive(Accounts)]
pub struct Refund<'info> {
    #[account(
        mut,
        seeds = [
            b"envelope",
            envelope.owner.as_ref(),
            &envelope.envelope_id.to_le_bytes()
        ],
        bump
    )]
    pub envelope: Account<'info, EnvelopeAccount>,

    #[account(mut)]
    pub owner: Signer<'info>,
}

// =========================
// CUSTOM ERRORS
// =========================

#[error_code]
pub enum CustomError {
    #[msg("Invalid owner")]
    InvalidOwner,

    #[msg("Already claimed by this address")]
    AlreadyClaimed,

    #[msg("Not allowed to claim this envelope")]
    NotAllowed,

    #[msg("Quota full - all claims taken")]
    QuotaFull,

    #[msg("Envelope has expired")]
    Expired,

    #[msg("Nothing to refund")]
    NothingToRefund,

    #[msg("Envelope amount exceeds maximum allowed")]
    ExceedMaxCreate,

    #[msg("Envelope not expired yet")]
    NotExpired,

    #[msg("Math overflow")]
    MathOverflow,

    #[msg("Insufficient funds in envelope")]
    InsufficientFunds,
}