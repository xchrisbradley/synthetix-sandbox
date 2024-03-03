const { exec } = require('child_process');

// Function to execute shell commands
const execPromise = (cmd: string) => new Promise((resolve, reject) => {
  exec(cmd, (error: any, stdout: string, stderr: any) => {
    if (error) {
      reject(error);
      return;
    }
    resolve(stdout.trim());
  });
});

// Main function to build and run
async function buildAndRun() {
  try {
    console.log('Building the project...');
    // Replace this command with your actual build command if different
    const buildOutput: any = await execPromise('npx @usecannon/cli build');
    console.log(buildOutput);

    // Extract the version from the build output
    const versionMatch = buildOutput.match(/synthetix-sandbox:(\d+\.\d+\.\d+@main)/);
    if (!versionMatch) {
      throw new Error('Unable to extract version from build output.');
    }
    const version = versionMatch[1];

    // Construct and execute the run command with the extracted version
    console.log(`Running synthetix-sandbox version ${version}...`);
    await execPromise(`cannon synthetix-sandbox:${version}`);
  } catch (error) {
    console.error('An error occurred:', error);
  }
}

// Execute the main function
buildAndRun();
