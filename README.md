Kurtosis Ethereum Quickstart
============================
The instructions below will walk you through spinning up an Ethereum network in a Kurtosis sandbox, interacting with it, and migrating the logic into the Kurtosis testing framework. By the end of this tutorial, you will have a rudimentary Ethereum testsuite in Typescript that you can begin to modify on your own.


Step One: Set Up Prerequisites (2 minutes)
------------------------------------------
### Install Docker
Verify that you have the Docker daemon installed and running on your local machine by running (you can copy this code by hovering over it and clicking the clipboard in the top-right corner):

```
docker image ls
```

* If you don't have Docker installed, do so by following [the installation instructions](https://docs.docker.com/get-docker/)
* If Docker is installed but not running, start it

**NOTE:** [DockerHub restricts downloads from users who aren't logged in](https://www.docker.com/blog/what-you-need-to-know-about-upcoming-docker-hub-rate-limiting/) to 100 images downloaded per 6 hours, so if at any point in this tutorial you see the following error message:

```
Error response from daemon: toomanyrequests: You have reached your pull rate limit. You may increase the limit by authenticating and upgrading: https://www.docker.com/increase-rate-limit
```

you can fix it by creating a DockerHub account (if you don't have one already) and registering it with your local Docker engine like so:

```
docker login
```

### Install the Kurtosis CLI
Follow the steps [on this installation page][installation] to install the CLI for your architecture & package manager.

Step Two: Start A Sandbox Enclave (3 minutes)
---------------------------------------------
The Kurtosis engine provides you isolated environments called "enclaves" to run your services inside. Let's use the CLI to start a sandbox enclave:

```
mkdir /tmp/my-enclave
cd /tmp/my-enclave
kurtosis sandbox
```

The Kurtosis images that run the engine will take a few seconds to pull the first time, but once done you'll have a Javascript REPL with tab-complete attached to your enclave.

All interaction with a Kurtosis enclave is done via [a client library][core-documentation], whose entrypoint is the `NetworkContext` object - a representation of the network running inside the enclave. The `networkCtx` variable in your REPL is how you'll interact with your enclave.

Let's check the contents of our enclave (this entire block can be copy-pasted as-is into the REPL):

```javascript
getServicesResult = await networkCtx.getServices()
services = getServicesResult.value
```

We haven't started any services yet, so the enclave will be empty. Note how we called `await` on `networkCtx.getServices()`. This is because every `networkCtx` call is asynchronous and returns a [Promise](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Promise); `await`ing blocks until that value is available.


Step Three: Start An Ethereum Network (5 minutes)
-------------------------------------------------
Now that we have an enclave, let's put something in it! Ethereum is one of the most popular blockchains in the world, so let's get a private Ethereum network running:

```javascript
loadEthModuleResult = await networkCtx.loadModule("eth-module", "kurtosistech/ethereum-kurtosis-module", "{}")
ethModuleCtx = loadEthModuleResult.value
executeEthModuleResult = await ethModuleCtx.execute("{}")
executeEthModuleResultObj = JSON.parse(executeEthModuleResult.value)
console.log(executeEthModuleResultObj)
```

This will take approximately a minute to run, with the majority of the time spent pulling the Ethereum images. After the final `console.log` line executes, you'll see a result with information about the services running inside your enclave:

```javascript
{
  bootnode_service_id: 'bootnode',
  node_info: {
    bootnode: {
      ip_addr_inside_network: '14.93.192.7',
      exposed_ports_set: [Object],
      port_bindings_on_local_machine: [Object]
    },
    'ethereum-node-1': {
      ip_addr_inside_network: '14.93.192.9',
      exposed_ports_set: [Object],
      port_bindings_on_local_machine: [Object]
    },
    'ethereum-node-2': {
      ip_addr_inside_network: '14.93.192.11',
      exposed_ports_set: [Object],
      port_bindings_on_local_machine: [Object]
    }
  },
  signer_keystore_content: '{"address":"14f6136b48b74b147926c9f24323d16c1e54a026","crypto":{"cipher":"aes-128-ctr","ciphertext":"39fb1d86c1082c0103ece1c5f394321f127bf1b65e6c471edcfb181058a3053a","cipherparams":{"iv":"c366d1eed33e8693fec7a85fad65d19f"},"kdf":"scrypt","kdfparams":{"dklen":32,"n":262144,"p":1,"r":8,"salt":"f210bc3b55117197f62a7ab8d85f2172342085f1daafa31034016163b8bc7db6"},"mac":"2ff8aa24d9b73ccfdb99cfd15fcdbcc8f640aaa7861e6813d53efaf550725fac"},"id":"6c5ac271-d24a-4971-b365-49490cc4befc","version":3}',
  signer_account_password: 'passphrase'
}
```

And if we query the enclave's services again...

```javascript
getServicesResult = await networkCtx.getServices()
services = getServicesResult.value
```

...we see three service IDs:

```javascript
Set(3) { 'bootnode', 'ethereum-node-1', 'ethereum-node-2' }
```

But what just happened?

Starting networks is a very common task in Kurtosis, so we provide [a framework called "modules"](https://docs.kurtosistech.com/modules.html) for making it dead simple. An executable module is basically a chunk of code that responds to an "execute" command, packaged as a Docker image, that runs inside a Kurtosis enclave - sort of like Docker Compose on steroids. In the steps above, we called `networkCtx.loadModule` to load [the Ethereum module](https://github.com/kurtosis-tech/ethereum-kurtosis-module) into the enclave with module ID `eth-module`, and `ethModuleCtx.execute` to run it. The Ethereum module doesn't take any parameters at load or execute time (hence the `{}`), but other modules do.

Now that you have a pet Ethereum network, let's do something with it.

Step Four: Talk To Ethereum (5 minutes)
---------------------------------------
Talking to Ethereum in Javascript is easily accomplished with [the EthersJS library](https://docs.ethers.io/v5/). Your Javascript REPL is running in a Docker image (so that you don't need Javascript installed locally), so we'll need to install EthersJS on that image.

First, in a new terminal window, run the following to find the enclave our REPL is running inside:

```
kurtosis enclave ls
```

You should see an output similar (but not identical) to the following:

```
EnclaveID
KT2021-10-17T15.46.23.438
```

Copy the enclave ID, and slot it into `YOUR_ENCLAVE_ID_HERE` in the below command:

```
kurtosis enclave inspect YOUR_ENCLAVE_ID_HERE
```

Kurtosis will output everything it knows about your enclave, similar but not identical to the output below:

```
======================================== Interactive REPLs ========================================
GUID
1634503584

========================================== User Services ==========================================
GUID                         LocalPortBindings
bootnode_1634503610          30303/udp -> 0.0.0.0:54291
                             8545/tcp -> 0.0.0.0:52113
                             8546/tcp -> 0.0.0.0:52111
                             30303/tcp -> 0.0.0.0:52112
ethereum-node-1_1634503612   30303/udp -> 0.0.0.0:59350
                             8545/tcp -> 0.0.0.0:52115
                             8546/tcp -> 0.0.0.0:52116
                             30303/tcp -> 0.0.0.0:52114
ethereum-node-2_1634503614   30303/udp -> 0.0.0.0:55007
                             8545/tcp -> 0.0.0.0:52170
                             8546/tcp -> 0.0.0.0:52171
                             30303/tcp -> 0.0.0.0:52172
```

Copy the interactive REPL's GUID, and replace both `YOUR_REPL_GUID_HERE` and `YOUR_ENCLAVE_ID_HERE` in the below command with the appropriate values:

```
kurtosis repl install YOUR_ENCLAVE_ID_HERE YOUR_REPL_GUID_HERE ethers
```

When the command finishes, you can now use it in your CLI! (You can execute the next command in the interactive REPL that should be still open in the previous tab)

```javascript
const ethers = require("ethers")
```

Now let's get a connection to the node with service ID `bootnode` by getting a [JsonRpcProvider](https://docs.ethers.io/v5/api/providers/jsonrpc-provider/):

```javascript
bootnodeServiceId = executeEthModuleResultObj.bootnode_service_id
bootnodeIp = executeEthModuleResultObj.node_info[bootnodeServiceId].ip_addr_inside_network
bootnodeRpcProvider = new ethers.providers.JsonRpcProvider(`http://${bootnodeIp}:8545`);
```

Notice how we used the `executeEthModuleResultObj` object containing details about the Ethereum network, which we got from executing the module at the very beginning.

Finally, let's verify that our Ethereum network is producing blocks:

```javascript
blockNumber = await bootnodeRpcProvider.getBlockNumber()
if (blockNumber > 0) { console.log("All is well!"); }
```

And that's it! Anything doable in Ethers is now doable against your private Ethereum network running in your Kurtosis enclave.

To exit out of the REPL you can enter any of:

* Ctrl-D
* Ctrl-C, twice
* `.exit`

and Kurtosis will tear down the enclave and everything inside.

Step Five: Get An Ethereum Testsuite (5 minutes)
---------------------------------------------
Manually verifying against a sandbox network is nice, but it'd be great if we could take our logic and run it as part of CI. Kurtosis has a testing framework that allows us to do exactly that. 

Normally, we'd bootstrap a testsuite from [the Testsuite Starter Pack](https://github.com/kurtosis-tech/kurtosis-testsuite-starter-pack) and use [the same Kurtosis engine documentation][core-documentation] with [the testing framework documentation](https://docs.kurtosistech.com/kurtosis-testsuite-api-lib/lib-documentation) to customize it for our Ethereum usecase.

For the purposes of this onboarding though, we've gone ahead and created an Ethereum testsuite that's ready to go. Go ahead and clone it from [here](https://github.com/kurtosis-tech/onboarding-ethereum-testsuite) now, and we'll take a look around.

The first thing to notice is the `testsuite/Dockerfile`. Testsuites in Kurtosis are simply packages of tests bundled in Docker images, which the testing framework will instantiate to run tests.

The second thing to notice is the `testsuite/testsuite_impl/eth_testsuite.ts` file. This is where tests are defined, and this testsuite already has a single test - `basicEthTest`.

Now open `testsuite/testsuite_impl/basic_eth_test/basic_eth_test.ts`. You'll see that a test is really just a class with three function: `configure`, `setup`, and `run`. Like most testing frameworks, `setup` is where we place the prep work that executes before the `run` method while `run` is where we make our test assertions. The `configure` method is where timeouts for both `setup` and `run` are configured, among other things.

The last thing to notice is how a `NetworkContext` is passed in as an argument to `setup`. Every Kurtosis test runs inside of its own enclave to prevent cross-test interference, and you can use [the exact same `NetworkContext` APIs][core-documentation] inside the testing framework that you used in the sandbox.

Now let's see the testing framework in action. From the root of the repo, run:

```
scripts/build-and-run.sh all    # The 'all' tells Kurtosis to build your testsuite into a Docker image AND run it
```

You'll see a prompt to create a Kurtosis account, which we use for gating advanced features (don't worry, we won't sign you up for any email lists!). Follow the instructions, and click the device verification link once you have your account.

The testsuite will run, and you'll see that our `basicEthTest` passed!

Step Six: Test Ethereum (5 minutes)
-----------------------------------
We now have a test running in the testing framework, but our test doesn't currently do anything. Let's fix that.

First, inside the `BasicEthTest` class, replace the `// TODO Replace with Ethereum network setup` line in the `setup` method with the following code:

```typescript
log.info("Setting up Ethereum network...")
const loadEthModuleResult: Result<ModuleContext, Error> = await networkCtx.loadModule(ETH_MODULE_ID, ETH_MODULE_IMAGE, "{}");
if (loadEthModuleResult.isErr()) {
    return err(loadEthModuleResult.error);
}
const ethModuleCtx: ModuleContext = loadEthModuleResult.value;

const executeEthModuleResult: Result<string, Error> = await ethModuleCtx.execute("{}")
if (executeEthModuleResult.isErr()) {
    return err(executeEthModuleResult.error);
}
this.executeEthModuleResultObj = JSON.parse(executeEthModuleResult.value);
log.info("Ethereum network set up successfully");
```

This is the same code we already executed in the REPL, cleaned up for Typescript. The only new bits to pay attention to are the error-checking: all `NetworkContext` methods, as well as the `Test.setup` and `Test.run` methods, return [a Result object][neverthrow] (much like in Rust). If `setup` or `run` return a non-`Ok` result, the test will be marked as failed. This allows for easy, consistent error-checking: simply propagate the error upwards.

Second, replace the `// TODO Replace with block number check` line with this code:

```typescript
log.info("Verifying block number is increasing...");
const bootnodeServiceId: ServiceID = this.executeEthModuleResultObj.bootnode_service_id;
const bootnodeIp: string = this.executeEthModuleResultObj.node_info[bootnodeServiceId].ip_addr_inside_network
const bootnodeRpcProvider: ethers.providers.JsonRpcProvider = new ethers.providers.JsonRpcProvider(`http://${bootnodeIp}:8545`);
const blockNumber: number = await bootnodeRpcProvider.getBlockNumber();
if (blockNumber === 0) {
    return err(new Error(""))
}
log.info("Verified that block number is increasing");
```

Finally, build and run the testsuite again:

```
scripts/build-and-run.sh all
```

You'll see logs like:

```
Setting up Ethereum network...
Ethereum network set up successfully
```

and

```
Verifying block number is increasing...
Verified that block number is increasing
```

indicating that our test set up an Ethereum network and ran our block count verification logic against it!

<!-- explain static files, and show how they could be used for ETH genesis -->
<!-- TODO Link to docs and further deepdives -->

<!-- TODO explain extra flags to control testsuite execution -->
<!-- TODO explain executing the testsuite in CI -->
<!-- TODO explain Debug mode, host port bindings, and setting debug log level -->

[installation]: https://docs.kurtosistech.com/installation.html
[neverthrow]: https://www.npmjs.com/package/neverthrow
[core-documentation]: https://docs.kurtosistech.com/kurtosis-client/lib-documentation
