import Foundation
import Security

struct KeychainService {
    private static let service = "com.strubio.MarvinTimeTracker"

    static var apiKey: String? {
        get { getString(account: "apiKey") }
        set { setString(newValue, account: "apiKey") }
    }

    static var serverURL: String? {
        get { getString(account: "serverURL") }
        set { setString(newValue, account: "serverURL") }
    }

    static var isConfigured: Bool {
        apiKey != nil && serverURL != nil
    }

    // MARK: - Private

    private static func getString(account: String) -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
        ]

        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)

        guard status == errSecSuccess, let data = result as? Data else {
            return nil
        }
        return String(data: data, encoding: .utf8)
    }

    private static func setString(_ value: String?, account: String) {
        if let value {
            let data = Data(value.utf8)

            let updateQuery: [String: Any] = [
                kSecClass as String: kSecClassGenericPassword,
                kSecAttrService as String: service,
                kSecAttrAccount as String: account,
            ]
            let updateAttributes: [String: Any] = [
                kSecValueData as String: data,
            ]

            let updateStatus = SecItemUpdate(updateQuery as CFDictionary, updateAttributes as CFDictionary)

            if updateStatus == errSecItemNotFound {
                var addQuery = updateQuery
                addQuery[kSecValueData as String] = data
                addQuery[kSecAttrAccessible as String] = kSecAttrAccessibleAfterFirstUnlock
                SecItemAdd(addQuery as CFDictionary, nil)
            }
        } else {
            let deleteQuery: [String: Any] = [
                kSecClass as String: kSecClassGenericPassword,
                kSecAttrService as String: service,
                kSecAttrAccount as String: account,
            ]
            SecItemDelete(deleteQuery as CFDictionary)
        }
    }
}
