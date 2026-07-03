import LocalAuthentication
import os.log

private let logger = Logger(subsystem: "com.eriknielsen.lockpaw", category: "Authenticator")

enum AuthenticationResult {
    case success
    case failed
    case cancelled
}

@MainActor
class Authenticator {
    private var activeContext: LAContext?

    /// Authenticate with Touch ID, with password fallback via system dialog.
    func authenticate(reason: String = "Unlock Lockpaw") async -> AuthenticationResult {
        cancelPending()

        let context = LAContext()
        context.localizedCancelTitle = "Cancel"
        context.localizedFallbackTitle = "Use Password\u{2026}"
        activeContext = context

        defer { activeContext = nil }

        var error: NSError?
        guard context.canEvaluatePolicy(.deviceOwnerAuthentication, error: &error) else {
            logger.error("Auth not available: \(error?.localizedDescription ?? "unknown")")
            return .failed
        }

        // Evaluate off MainActor to avoid deadlock — system dialog needs main thread
        return await Task.detached { [context] in
            do {
                let authenticated = try await context.evaluatePolicy(
                    .deviceOwnerAuthentication,
                    localizedReason: reason
                )
                return authenticated ? .success : .failed
            } catch {
                let result = Self.result(for: error)
                await MainActor.run {
                    logger.info("Auth cancelled or failed: \(error.localizedDescription)")
                }
                return result
            }
        }.value
    }

    /// Authenticate with macOS password (system dialog, user can click "Use Password").
    func authenticateWithPassword(reason: String = "Enter your password to unlock Lockpaw") async -> AuthenticationResult {
        cancelPending()

        let context = LAContext()
        context.localizedCancelTitle = "Cancel"
        context.localizedFallbackTitle = ""
        activeContext = context

        defer { activeContext = nil }

        var error: NSError?
        guard context.canEvaluatePolicy(.deviceOwnerAuthentication, error: &error) else {
            logger.error("Password auth not available: \(error?.localizedDescription ?? "unknown")")
            return .failed
        }

        return await Task.detached { [context] in
            do {
                let authenticated = try await context.evaluatePolicy(
                    .deviceOwnerAuthentication,
                    localizedReason: reason
                )
                return authenticated ? .success : .failed
            } catch {
                let result = Self.result(for: error)
                await MainActor.run {
                    logger.info("Password auth cancelled or failed: \(error.localizedDescription)")
                }
                return result
            }
        }.value
    }

    func cancelPending() {
        activeContext?.invalidate()
        activeContext = nil
    }

    private nonisolated static func result(for error: Error) -> AuthenticationResult {
        guard let laError = error as? LAError else { return .failed }
        switch laError.code {
        case .userCancel, .systemCancel, .appCancel:
            return .cancelled
        default:
            return .failed
        }
    }
}
