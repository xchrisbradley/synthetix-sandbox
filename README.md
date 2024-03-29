# Synthetix Sandbox

This is a boilerplate for developing front-ends and smart contract integrations for [Synthetix V3](https://github.com/Synthetixio/synthetix-v3).

Install [Cannon](https://usecannon.com) with `npm i -g @usecannon/cli` and review the [Get Started guide](https://usecannon.com/learn/guides/get-started).

**⚠️ Remember to always interact with the proxy contracts instead of the router or modules directly.**

## Developing Front-ends

- Run `cannon synthetix-sandbox` to start a local node as defined in the [Cannonfile](/cannonfile.toml) for front-end development.
- Run `cannon inspect synthetix-sandbox --write-deployments ./deployments` to export the smart contract ABIs and addresses.

## Developing Smart Contract Integrations

- Fork this repository.
- Modify the [Sample Integration contract](/src/SampleIntegration.sol).
- Customize the name at the top of `cannonfile.toml` and make any other modifications.
- Make sure you have the Synthetix Router Cannon plug-in installed: `npx cannon plugin add cannon-plugin-router`
- Run `cannon build` to create a local build of your Cannonfile.
- Run `cannon <replace-with-new-cannonfile-name>` to start it on a local node.

See the [Production Cannonfile](/cannonfile.prod.toml) for an example Cannonfile that deploys the Sample Integration contract integrated with the official [Synthetix V3 Deployments](https://github.com/Synthetixio/synthetix-deployments).

+++++

---
title: "How to Build a Custom Market on Synthetix V3"
subtitle: "Get started by creating a simple number guessing game"
date: 2023-04-07
tags: ["defi", "synthetix", "howto"]
draft: true
---

Earlier this year, Synthetix deployed V3, its next generation derivatives platform, to mainnet. A culmination of years of research and experience in creating on-chain derivatives products, the latest version of the protocol consists of a suite of systems and modules. This design allows developers to build custom market implementations for a wide range of financial instruments, such as spot tokens, perpetual futures, options, insurance, and any else, while utilizing the highest degree of decentralized technology currently available.

This is a guide to building a simple number guessing game, implemented as a market on Synthetix V3.

Before starting, you will need:

- Some [Solidity](https://solidity-by-example.org/) software engineering experience.
- Access to a JSON RPC URL for the networks you want to deploy on. (This guide uses the Sepolia testnet.)
- Access to a private key with a bit of ETH on the networks you want to deploy on.
- [An IPFS node](https://docs.ipfs.tech/install/ipfs-desktop/) running in your local development environment.

## What are we building?

Synthetix V3 allows developers to bootstrap liquidity for derivatives markets by incentivizing liquidity providers. To demonstrate how this works, we'll build a simple number guessing game.

Typically, a number guessing game implemented using a smart contract could not provide substantial prizes or consistent odds without first accumulating an adequate prize pool from numerous players. If it were to offer consistent odds or high payouts prematurely, the number guessing game would risk insolvency (i.e. be unable to pay out winners). Consequently, the initial players would be uncertain about the size of the potential prize and their odds of winning, so it would be hard to attract early players to address the first issue. This scenario exemplifies the "cold start liquidity" problem, which is relevant to all types of derivatives markets.

A number guessing game implemented with Synthetix can offer liquidity providers the ability to collect fees from ticket sales in exchange for providing collateral to be used in prize payouts if necessary. In this guide, the number guessing game will offer tickets for 1 USD, and have a consistent prize of 1,000 USD. These numbers could be configured differently, or even set dynamically as a function of other factors. Let your imagination run wild.

There will be three main methods in the smart contract for the market:

- `buyTicket()`: In exchange for the ticket price, this function gives the user a chance to win the prize at the next draw.
- `draw()`: This function calls Chainlink to request a random number. Chainlink will return the random number in a call to `payout()`.
- `payout()`: If a participant previously called `buyTicket` with the matching number, they win the prize. Otherwise, all proceed from ticket sales are automatically distributed among liquidity providers by the Synthetix system.

In a naive implementation where a constant prize were provided, a user could buy one ticket, call `draw()`, and immediately win 1,000 USD. Obviously, this would not be an appealing market for liquidity providers to back, as the market could provide the player with risk-free yield at their expense.

To make the market profitable for liquidity providers, the odds of any individual ticket winning could be 1 in 1,000 and a fee that is always earned to liquidity providers could be added.

Picking random numbers in Solidity is hard. To put it simply, a miner of a block can effectively manipulate all of the data in that block, so it is impossible to generate a truly random number without the use of some external service. In order to solve this problem, the number guessing game will utilize [Chainlink VRF](https://chain.link/vrf). Using a pool of distributed and anonymous nodes, Chainlink is able to generate random numbers off-chain and provide them directly to a consumer contract on-chain. The best part is, any numbers generated by this service can be cryptographically verified for fairness, so it is safe to use as part of the implementation.

Once initialized, the market contract will not require any parameter modifications, so it will be immutable and permissionless upon deployment. This design ensures optimal security and transparency, as no individual or entity can control any aspect of the game, effectively eliminating the possibility of undue influence.

Finally, every smart contract could use a nice user interface. This guide will show how to build a very simple web application that elegantly handles cross-chain considerations and integrates with with development tooling.

## Setup

For this guide we will be building the market with [Foundry](). Even if you prefer to do the project with Hardhat, use of later tools requires Anvil from Foundry. If you haven't already, you can install Foundry to your machine:

```bash
curl -L https://foundry.paradigm.xyz | bash
foundryup
```

## Initializing the Project

Foundry provides a command for initializing an empty project. To start, create an empty directory, and then run the initialization command:

```bash
mkdir number-guessing-game
cd number-guessing-game
forge init
```

Delete the provided sample `Counter.sol` and `Counter.t.sol` files, as we will be replacing them with our own instead:

```bash
rm src/Counter.sol test/Counter.t.sol
```

[Chainlink VRF (verified randomness framework)](https://chain.link/vrf) is used to generate the winning number. This is available in the main Chainlink monorepo:

```
forge install smartcontractkit/chainlink --no-commit
```

## Implementing the Market Interface

All markets in Synthetix V3 must implement the [IMarket](https://github.com/Synthetixio/synthetix-v3/blob/main/protocol/synthetix/contracts/interfaces/external/IMarket.sol) interface. Save this file to `src/external/IMarket.sol`. Replace the IERC165 import in this file with: `import "lib/forge-std/src/interfaces/IERC165.sol";`

To generate an interface for the Synthetix V3 Core System, we can download the ABI using [Cannon](https://usecannon.com) and generate an interface file using [abi-to-sol](https://github.com/gnidan/abi-to-sol). (Make sure you have an [an IPFS node](https://docs.ipfs.tech/install/ipfs-desktop/) running to access the ABIs.)

```
npm install -g @usecannon/cli abi-to-sol
cannon inspect --chain-id 10 synthetix:latest --json | jq '.state["router.CoreRouter"].artifacts.contracts.CoreRouter.abi' -cM | abi-to-sol ISynthetixCore -V '^0.8.4' > src/external/ISynthetixCore.sol
```

Now open your favorite code editor and create a file `src/NumberGuessingGame.sol`. Lets put in the minimum contents of a market contract that implements the IMarket interface and imports the files we just brought into the project:

```js
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import "./external/IMarket.sol";
import "./external/IMarketManagerModule.sol";

import "lib/forge-std/src/interfaces/IERC20.sol";
import "lib/chainlink/contracts/src/v0.8/VRFV2WrapperConsumerBase.sol";

contract NumberGuessingGame is VRFV2WrapperConsumerBase, IMarket {
    function name(uint128 _marketId) external override view returns (string memory n) {
        if (_marketId == marketId) {
            n = string(abi.encodePacked("Market ", bytes32(uint256(_marketId))));
        }
    }

    function reportedDebt(uint128) external override pure returns (uint256) {
        return 0;
    }

    function minimumCredit(uint128 _marketId) external override view returns (uint256) {
        return 0;
    }

    function supportsInterface(
        bytes4 interfaceId
    ) public view virtual override(IERC165) returns (bool) {
        return
            interfaceId == type(IMarket).interfaceId ||
            interfaceId == this.supportsInterface.selector;
    }
}
```

As you can see, other than `supportsInterface` for ERC-165 compatability, we have to implement three functions. Let's go over these in detail:

- `name(uint128 marketId) returns (string memory)`: Returns a human-readable name for the market. This is useful for display on dashboards or when people are browsing markets registered with Synthetix. In the code above, we are rendering the ID as a bytes string, which isn't ideal. You could improve this code by relying on Open Zeppelin’s [string library](https://docs.openzeppelin.com/contracts/4.x/api/utils#Strings).
- `reportedDebt(uint128 marketId) returns (uint256)`: Allows for a market to share the amount of _unrealized debt_ which should be distributed to the liquidity providers for this market. For example, a spot token market would return `totalSupply * tokenPrice` because, if all those tokens were sold, that is the amount of stablecoins which would need to be paid out. Liquidity providers delegating to this market through a pool will effectively take on the debt reported by this function. For the number guessing game, this value will always be `0` since the contract will not be holding any unrealized debt; all debt is immediately realized as soon as the contract pays out.
- `minimumCredit(uint128 marketId) returns (uint256)`: Similar to `reportedDebt`, this allows the market to control the minimum amount of liquidity provided to it via pools. Collateral cannot be withdrawn such that the remaining credit capacity is below the amount returned by this function. This is useful when a market may be about to accumulate a large amount of debt and liquidity providers might otherwise be able to leave in anticipation. The number guessing game uses this function to prevent withdrawals when a draw is in progress, as we will see later.

## Implementing the Smart Contract Functions

Let's start by adding some variables to the top of the contract:

```js
    ISynthetixCore public synthetix; // Address of the Synhtetix core system
    IERC20 public linkToken; // Address of the LINK token
    uint128 public marketId; // Market ID, assigned by the Synthetix Core system

    uint256 public prize; // Payout amount, denominated in USD with 18 decimals places,
    uint256 public ticketCost; // Cost of the ticket, denominated in USD with 18 decimals places,
    uint256 public feePercent; // Percentage of ticket cost to collect for LPs. 1 followed by 18 zeros represents 100%

    uint256 private currentDrawRound; // The current draw round, for referencing ticketBuckets
    bool private isDrawing; // Whether the market is waiting on the Chainlink VRF callback to payout the round

    mapping(uint256 => mapping(uint256 => address[])) ticketBuckets; // A mapping of draw rounds to a mapping of ticket numbers to an array of addresses that have purchased tickets for them.
    mapping(uint256 => uint256) requestIdToRound; // A mapping of request IDs (for Chainlink VRF) to draw rounds
```

Now let's add a constructor method to initialize the contract and an external method to register the market with the Synthetix core system:

```js
    constructor(
        ISynthetixCore _synthetix,
        address link,
        address vrf,
        uint256 _prize,
        uint256 _ticketCost,
        uint256 _feePercent
    ) VRFV2WrapperConsumerBase(link, vrf) {
        synthetix = _synthetix;
        linkToken = IERC20(link);
        prize = _prize;
        ticketCost = _ticketCost;
        feePercent = _feePercent;
    }

    function registerMarket() external {
        if (marketId == 0) {
            marketId = synthetix.registerMarket(address(this));
            emit MarketRegistered(marketId);
        }
    }
```

Now we’ll add the `buy` function with a `getMaxBucketParticipants` helper function to the `NumberGuessingGame` contract:

```js
    error InsufficientLiquidity(uint256 guessNumber, uint256 maxParticipants);

    function buy(address beneficary, uint guessNumber) external {
        address[] storage bucketParticipants = ticketBuckets[currentDrawRound][guessNumber % _bucketCount()];

        uint maxParticipants = getMaxBucketParticipants();

        if (bucketParticipants.length >= maxParticipants) {
            revert InsufficientLiquidity(guessNumber, maxParticipants);
        }

        IERC20(synthetix.getUsdToken()).transferFrom(msg.sender, address(this), ticketCost);
        bucketParticipants.push(beneficary);
    }

    function getMaxBucketParticipants() public view returns (uint256) {
        return synthetix.getWithdrawableMarketUsd(marketId) / prize;
    }
```

This function is mostly self-explanatory. However, we do need to add a check to limit the market's risk exposure with `maxParticipants`. This prevents a case where 10 users all pick number `42`. If Chainlink VRF draws `42`, but the market only has 5000 USD of available liquidity, it would go insolvent (i.e. be unable to pay the winners the amount they deserve).

We also need to create `startDraw` and `finishDraw` to allow for winners to be selected, with a few helper functions:

```js
    error DrawAlreadyInProgress();

    function startDraw(uint256 maxLinkCost) external {
        if (isDrawing) {
            revert DrawAlreadyInProgress();
        }

        // because of the way chainlink's VRF contracts work, we must transfer link from the sender before continuing
        linkToken.transferFrom(msg.sender, address(this), maxLinkCost);

        // initialize the request for a random number, transfer LINK from the sender's account
        uint256 requestId = requestRandomness(
            500000, // max callback gas
            0, // min confirmations
            1 // number of random values
        );

        requestIdToRound[requestId] = currentDrawRound++;

        isDrawing = true;
    }

    function finishDraw(uint256 round, uint256 winningNumber) internal {
        address[] storage winners = ticketBuckets[round][winningNumber % _bucketCount()];

        // if we dont have sufficient deposits, withdraw stablecoins from LPs
        IERC20 usdToken = IERC20(synthetix.getUsdToken());
        uint currentBalance = usdToken.balanceOf(address(this));
        if (currentBalance < prize * winners.length) {
            synthetix.withdrawMarketUsd(
                marketId,
                address(this),
                prize * winners.length - currentBalance
            );

            currentBalance = prize * winners.length;
        }

        // now send the deposits
        for (uint i = 0;i < winners.length;i++) {
            usdToken.transfer(winners[i], prize);
        }

        // update what our balance should be
        currentBalance -= prize * winners.length;

        // send anything remaining to the deposit
        if (currentBalance > 0) {
            synthetix.depositMarketUsd(marketId, address(this), currentBalance);
        }

        // allow for the next draw to start and unlock funds
        isDrawing = false;
    }

    function fulfillRandomWords(uint256 requestId, uint256[] memory randomWords) internal override virtual {
        finishDraw(requestIdToRound[requestId], randomWords[0]);
    }

    function _bucketCount() internal view returns (uint256) {
        uint256 baseBuckets = prize / ticketCost;
        return baseBuckets + baseBuckets * feePercent;
    }
```

In order to initialize a request with Chainlink VRF, LINK tokens must be provided by the caller in order to cover the costs of the draw. You could also implement the market could to cover this cost automatically (e.g. through withdrawing stablecoins and buying LINK through a decentralized exchange).

`finishDraw()` is called (via the `fulfillRandomWords` function) by the Chainlink oracles with the random number as requested. Th finish draw function distributes the prize to the users who won the draw.

If there are not enough stablecoins sitting in the market contract to cover this, the contract withdraws more stablecoins. If excess stablecoins remain in the contract after distributing the prizes (if any), the stablecoins are deposited, automatically distributing them to LPs.

By doing this, the LPs backing the market are penalized or rewarded based on the performance of the market. We would expect stakers to statistically profit 1 cent for every $1 ticket purchased (based on the 1% fee we have set above) on average.

If you run `forge build`, you should find that the code compiles without error.

## Testing

We will now build some tests to make sure that the basic functionality of the number guessing game is working as intended. Normally, we would have to either mock Synthetix and Chainlink VRF or figure out how to deploy them. Instead, Synthetix uses a tool called [Cannon](https://usecannon.com) which makes this process much easier my managing deployments to local, test, and production blockchains.

We'll start by creating a Cannonfile that imports Synthetix and Chainlink VRF. It will deploy the smart contract and call the `registerMarket` function:

```
name = "number-guessing-game"
version = "0.1.0"
description = "Demo market for Synthetix V3"

[setting.prize]
defaultValue = "1000000000000000000000"

[setting.ticketCost]
defaultValue = "1000000000000000000"

[setting.feePercent]
defaultValue = "10000000000000000"

[setting.salt]
defaultValue = "snax"

[import.vrf]
source = "chainlink-vrf:2.0.0"

[import.synthetix]
source = "synthetix:3.0.4-alpha.0"

[contract.NumberGuessingGame]
artifact = "NumberGuessingGame"
create2 = true
args = [
    "<%= imports.synthetix.contracts.CoreProxy.address %>",
    "<%= imports.vrf.imports.linkAggregator.imports.linkToken.contracts.Token.address %>",
    "<%= imports.vrf.contracts.VRFWrapper.address %>",
    "<%= settings.prize %>",
    "<%= settings.ticketCost %>",
    "<%= settings.feePercent %>"
]
depends = ["import.vrf", "import.synthetix"]

[invoke.registerMarket]
target = ["NumberGuessingGame"]
func = "registerMarket"
extra.marketId.event = "MarketRegistered"
extra.marketId.arg = 0
depends = ["contract.NumberGuessingGame"]
```

Next, we’ll create a cannonfile for tests that extends this one. (TODO: Additional explainer here.)

```
include = [
    "cannonfile.toml"
]

[import.sandbox]
source = "synthetix-sandbox:latest"

[import.synthetix]
source = "synthetix:latest"
preset = "with-synthetix-sandbox"
depends = ["import.sandbox"]

[invoke.setCollateralConfig]
target = ["synthetix.CoreProxy"]
fromCall.func = "owner"
func = "setPoolConfiguration"
args = [
    1,
    [
        { marketId = '<%= extras.marketId %>', weightD18 = '1', maxDebtShareValueD18 = "<%= parseEther('1').toString() %>" }
    ]
]
depends = ["invoke.registerMarket"]
```

Next, to integrate Cannon with Foundry, the `cannon-std` library must be imported for use in the tests:

```
forge install usecannon/cannon-std
```

Now we are ready to implement our tests! Create a file at `test/NumberGuessingGame.t.sol` and bring over the code from [this repository](), modifying as you see fit.

To run the tests with injected dependencies and full environment:

```
cannon test cannonfile.test.toml
```

What exactly is happening here?

- Cannon uses the `cannonfile.test.toml` to generate the exact deployment state of the number guessing game on your local network.
- It saves the addresses to JSON files in your `deployments` directory so that they can be queried within Foundry.
- It executes Foundry on the network that Cannon just created with `forge test`, and the tests get the address from `Cannon.getAddress()` library function.

That's it! No more deployment scripts, complicated test setups, or problems with mocks that diverge from real functionality. As you can see in the next section, this `cannonfile.toml` can be reused to handle the deployment to testnets or mainnets.

## Deployment

### Simulate a Deployment

Now that we have a smart contract and have written tests for it, we can deploy it to a testnet. As Cannon is a deployment and packaging tool, it can also be used for the task of deploying newly built markets. Deployment of the market is the same as a regular Cannon build, but you have to specify a remote RPC endpoint (and, most likely, a private key with ETH as the deployer). Additionally, its a good idea to simulate the release before actually running it.

Another nice feature of Cannon is that any of your dependencies (such as Chainlink or Synthetix V3) will automatically resolve the addresses for the actual deployments on their networks, so you do not need to fuss with connecting the correct addresses to your contracts or making sure that you are using the correct network.

To simulate a release of the contract to Goerli, use a command like below:

```
cannon build --network $GOERLI_RPC --private-key $DEPLOYER_PRIVATE_KEY --dry-run
```

Assuming the output is as you would expect, remove `--dry-run` to perform an actual release:

```
cannon build --network $GOERLI_RPC --private-key $DEPLOYER_PRIVATE_KEY
```

## Manually Test

Now that we have deployed the contract to Goerli, we can verify that it is working using Cannon.

Cannon includes a built-in CLI which allows for you to select and call methods on a contract. It also decodes all relevant data, such as ABI names of arguments, or revert errors in the event of a failure. This CLI can be used either on a fork, or directly on the network. To launch the interact tool directly, run:

```
cannon interact number-guessing-game:1.0.0 --network $GOERLI_RPC
```

Select the contract to execute a function call on. Lets open the `NumberGuessingGame` contract. Inside, you will find all the external methods we defined for the number guessing game. Push enter on `name()`. You should see that the number guessing game returns the appropriate string. You could also try buying a ticket with `buy()`, but if your testing account doesn't have any stablecoins, the call will fail. A decoded explanation of the error should appear when you do this.

You can also run interact on a fork. This has the added benefit of being able to "impersonate" any address. For example:

```
cannon run number-guessing-game:1.0.0 --fork $GOERLI_RPC --impersonate 0x48914229deDd5A9922f44441ffCCfC2Cb7856Ee9
```

## Building a Simple UI

See the `ui` directory for an example user interface for the market contract. You can use whichever stack you’re most comfortable with, but this example repository relies on [Ethers](https://docs.ethers.org/v5/), [Next](https://nextjs.org/), [RainbowKit](https://www.rainbowkit.com/), and [wagmi](https://wagmi.sh/). Note that all of the ABIs are retrieved from the deployments folder generated by Cannon.
