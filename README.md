
# Web3Pro Cypto Wallet

This is a crypto wallet that I use for my personal use. It is a selfware project. It is not intended for mass consumption


# SELFWARE

- **SelfWare**: Software made for myself, shared openly.

- **Not a Product**: Never intended for mass consumption or to meet market needs.

- **Developer-Driven**: Built exactly the way the developer envisions—no compromises, no community influence.

- **Open Source**: Open for anyone to use, modify, or adapt as they see fit, while staying true to the developer's original vision.

- **Independence**: No roadmaps driven by user feedback, no polls, no votes—only the developer's decisions.

- **A Personal Tool**: Solves specific problems for the creator, rather than catering to a broad user base.



# Install Nerd Fonts

https://github.com/ryanoasis/nerd-fonts?tab=readme-ov-file


On linux:

download fro here: https://www.nerdfonts.com/font-downloads


copy to: ~/.local/share/fonts 

```
fc-cache -fv
fc-list | grep "Nerd"

```

Select the font for your terminal


# GoCui package

The code is using the modified version of the https://github.com/awesome-gocui/gocui

# Building Prerequisites

```
sudo apt install pkg-config libasound2-dev libusb-1.0-0-dev
```

On WSL or systems without AVX-512, build with portable mode to avoid SIGILL:

```
CGO_CFLAGS="-O -D__BLST_PORTABLE__" go run .
```

# API Keys

The wallet requires two API keys to function.

## CoinMarketCap (CMC)

Provides cryptocurrency price and market data.

1. Sign up at https://pro.coinmarketcap.com/signup (free tier available)
2. Get your API key from the dashboard
3. In web3pro, run (no quotes around the key):
   ```
   cfg set cmc_api_key sk-1234567890abcdef
   ```

## The Graph

Provides blockchain indexing for querying on-chain data (liquidity pools, etc.).

1. Go to https://thegraph.com/studio/apikeys/
2. Connect a wallet (MetaMask works via WalletConnect)
3. Create an API key
4. In web3pro, run (no quotes around the key):
   ```
   cfg set thegraph_api_key abc123def456
   ```
