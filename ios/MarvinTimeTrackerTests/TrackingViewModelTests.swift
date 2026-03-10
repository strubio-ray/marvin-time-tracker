import Testing
import Foundation
@testable import MarvinTimeTracker

// MARK: - Mocks

final class MockKeychainService: KeychainServiceProtocol {
    var apiKey: String?
    var serverURL: String?
    var isConfigured: Bool { apiKey != nil && serverURL != nil }
}

struct MockMarvinAPIClient: MarvinAPIClientProtocol {
    var todayItemsResult: [MarvinTask] = []
    var fetchStatusResult: ServerStatus?
    var shouldThrow = false

    func todayItems() async throws -> [MarvinTask] {
        if shouldThrow { throw URLError(.badServerResponse) }
        return todayItemsResult
    }

    func startTracking(taskId: String, title: String) async throws {
        if shouldThrow { throw URLError(.badServerResponse) }
    }

    func stopTracking(taskId: String?) async throws {
        if shouldThrow { throw URLError(.badServerResponse) }
    }

    func fetchStatus() async throws -> ServerStatus {
        if shouldThrow { throw URLError(.badServerResponse) }
        return fetchStatusResult ?? ServerStatus(
            status: "ok",
            tracking: false,
            taskId: nil,
            taskTitle: nil,
            startedAt: nil,
            hasPushToStartToken: false,
            hasUpdateToken: false
        )
    }
}

struct MockPushTokenService: PushTokenServiceProtocol {
    func register(pushToStartToken: String?, updateToken: String?, deviceToken: String?) async throws {}
}

// MARK: - Tests

@Suite("TrackingViewModel")
struct TrackingViewModelTests {
    @Test("Initial state is idle")
    @MainActor
    func initialState() {
        let keychain = MockKeychainService()
        let vm = TrackingViewModel(keychain: keychain)
        #expect(vm.trackingState == .idle)
        #expect(vm.todayTasks.isEmpty)
        #expect(!vm.isLoading)
    }

    @Test("saveCredentials updates keychain and onboarded state")
    @MainActor
    func saveCredentials() {
        let keychain = MockKeychainService()
        let vm = TrackingViewModel(keychain: keychain)
        #expect(!vm.isOnboarded)

        vm.saveCredentials(apiKey: "key123", serverURL: "https://example.com")
        #expect(keychain.apiKey == "key123")
        #expect(keychain.serverURL == "https://example.com")
        #expect(vm.isOnboarded)
    }

    @Test("signOut clears credentials and resets state")
    @MainActor
    func signOut() {
        let keychain = MockKeychainService()
        keychain.apiKey = "key"
        keychain.serverURL = "https://example.com"
        let vm = TrackingViewModel(keychain: keychain)

        vm.signOut()
        #expect(keychain.apiKey == nil)
        #expect(keychain.serverURL == nil)
        #expect(!vm.isOnboarded)
        #expect(vm.trackingState == .idle)
    }

    @Test("refreshStatus updates to tracking state")
    @MainActor
    func refreshStatusTracking() async {
        let keychain = MockKeychainService()
        keychain.apiKey = "key"
        keychain.serverURL = "https://example.com"

        let mockClient = MockMarvinAPIClient(
            fetchStatusResult: ServerStatus(
                status: "ok",
                tracking: true,
                taskId: "task-1",
                taskTitle: "My Task",
                startedAt: 1772734813781,
                hasPushToStartToken: false,
                hasUpdateToken: true
            )
        )

        let vm = TrackingViewModel(
            keychain: keychain,
            apiClientFactory: { _, _ in mockClient }
        )

        await vm.refreshStatus()
        if case .tracking(let taskId, let title, _) = vm.trackingState {
            #expect(taskId == "task-1")
            #expect(title == "My Task")
        } else {
            Issue.record("Expected tracking state")
        }
    }

    @Test("refreshStatus sets idle when not tracking")
    @MainActor
    func refreshStatusIdle() async {
        let keychain = MockKeychainService()
        keychain.apiKey = "key"
        keychain.serverURL = "https://example.com"

        let mockClient = MockMarvinAPIClient()
        let vm = TrackingViewModel(
            keychain: keychain,
            apiClientFactory: { _, _ in mockClient }
        )

        await vm.refreshStatus()
        #expect(vm.trackingState == .idle)
    }
}
