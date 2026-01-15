# Project Structure Overview

## Project Overview
This project, `summit`, is a command-line tool written in Go for declaratively managing Alpine Linux systems. It allows users to define the desired state of a system (packages, services, users, configuration files) in a YAML file. `summit` then infers the current state of the system and calculates and executes a series of actions to bring the system to the desired state. It features a transactional approach with rollback capabilities to ensure system stability.

## Key Directories and Files
*   `/cmd`: Contains the command-line interface logic. It uses the `cobra` library to define commands like `apply`, `diff`, and `dump`.
    *   `root.go`: Sets up the root `summit` command and global flags (e.g., for logging).
    *   `apply.go`: Implements the core `apply` command, which drives the process of converging the system to the desired state.
*   `/pkg`: Contains the core application logic, separated into several packages.
    *   `/pkg/actions`: Defines the `Action` interface and concrete action implementations (e.g., `FileCreateAction`, `PackageInstallAction`). Each action is a discrete, reversible operation.
    *   `/pkg/config`: Handles loading and parsing the `system.yaml` configuration file.
    *   `/pkg/diff`: Contains the logic for comparing the desired and current system states to generate a plan of actions.
    *   `/pkg/log`: Provides a simple logging interface.
    *   `/pkg/model`: Defines the data structures that represent the system state, as loaded from the YAML configuration.
    *   `/pkg/runner`: Implements the execution of the action plan, including the rollback mechanism.
    *   `/pkg/system`: Provides an abstraction layer for interacting with the underlying system (e.g., filesystem, command execution).
*   `/test`: Contains integration and end-to-end tests.
    *   `/test/integration`: Includes tests that run against a real or containerized Alpine Linux environment to verify the end-to-end functionality.
*   `main.go`: The main entry point of the application.
*   `go.mod`: The Go module file, which defines the project's dependencies.
*   `system.yaml`: The default configuration file where users define the desired state of the system.

## Architectural Patterns
`summit` follows a **declarative, state-driven architecture**. The core of the application is a reconciliation loop that performs the following steps:
1.  **Load Desired State:** The desired state of the system is loaded from a YAML file (`system.yaml`) into the `model.SystemState` struct.
2.  **Infer Current State:** The application inspects the live system to determine its current state.
3.  **Calculate Diff:** The `diff.CalculatePlan` function compares the desired and current states and produces a "plan," which is a sequence of `actions.Action` objects.
4.  **Execute Plan:** The `executePlan` function iterates through the actions in the plan and executes their `Apply` methods.

A key architectural feature is the **transactional nature of actions**. Every action has a corresponding `Rollback` method. If any action in the plan fails, the system executes the `Rollback` method for all previously completed actions in reverse order, leaving the system in its original state.

The use of the `afero` library for filesystem operations allows for easy testing by swapping out the real filesystem with an in-memory one.

## Getting Started & Code Flow
A new developer should start by understanding the data structures in `pkg/model/state.go`, as they are central to the application's logic.

The application's main logic flow is initiated by the `applyCmd` in `cmd/apply.go`. Here's a breakdown of the code flow:
1.  The `applyCmd`'s `RunE` function is executed when a user runs `summit apply`.
2.  `config.LoadConfig` is called to load the `system.yaml` file.
3.  `system.InferSystemState` is called to determine the current state of the system.
4.  `diff.CalculatePlan` is called to generate the list of actions to be executed.
5.  The `executePlan` function iterates through the generated plan, calling the `Apply` method on each action. If an error occurs, `rollbackPlan` is called to revert any changes.

To understand how `summit` modifies the system, a developer should examine the different `Action` implementations in the `pkg/actions/` directory. Each action is a self-contained unit of work that modifies a specific aspect of the system.
