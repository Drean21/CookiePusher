<template>
  <div class="min-h-screen bg-gray-100 text-gray-800">
    <header class="bg-white shadow">
      <nav class="container mx-auto px-6 py-4">
        <h1 class="text-2xl font-bold text-gray-700">Cookie Syncer - Admin Panel</h1>
      </nav>
    </header>

    <main class="container mx-auto px-6 py-8">
      <div class="bg-white p-6 rounded-lg shadow-lg">
        <h2 class="text-xl font-semibold mb-4">Search Cookies</h2>
        <form @submit.prevent="handleSearch" class="flex items-center space-x-4">
          <div class="flex-1">
            <label for="domain" class="block text-sm font-medium text-gray-600"
              >Domain</label
            >
            <input
              type="text"
              id="domain"
              v-model="searchDomain"
              class="mt-1 block w-full px-3 py-2 bg-white border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
              placeholder="e.g., google.com"
            />
          </div>
          <div class="flex-1">
            <label for="name" class="block text-sm font-medium text-gray-600"
              >Cookie Name</label
            >
            <input
              type="text"
              id="name"
              v-model="searchName"
              class="mt-1 block w-full px-3 py-2 bg-white border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
              placeholder="e.g., NID"
            />
          </div>
          <div class="pt-6">
            <button
              type="submit"
              class="px-4 py-2 bg-indigo-600 text-white font-semibold rounded-md shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
            >
              Search
            </button>
          </div>
        </form>
      </div>

      <div class="mt-8 bg-white p-6 rounded-lg shadow-lg">
        <h3 class="text-lg font-semibold mb-4">Results</h3>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200">
            <thead class="bg-gray-50">
              <tr>
                <th
                  class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                >
                  Domain
                </th>
                <th
                  class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                >
                  Name
                </th>
                <th
                  class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                >
                  Value
                </th>
                <th
                  class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                >
                  User ID
                </th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-gray-200">
              <tr v-if="loading">
                <td colspan="4" class="text-center py-4">Loading...</td>
              </tr>
              <tr v-else-if="error">
                <td colspan="4" class="text-center py-4 text-red-500">{{ error }}</td>
              </tr>
              <tr v-else-if="results.length === 0">
                <td colspan="4" class="text-center py-4">No results found.</td>
              </tr>
              <tr v-for="cookie in results" :key="cookie.id">
                <td class="px-6 py-4 whitespace-nowrap">{{ cookie.domain }}</td>
                <td class="px-6 py-4 whitespace-nowrap">{{ cookie.name }}</td>
                <td
                  class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 truncate"
                  :title="cookie.value"
                >
                  {{ cookie.value }}
                </td>
                <td class="px-6 py-4 whitespace-nowrap">{{ cookie.user_id }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";

// Mock data structure, will be replaced with real API calls
interface CookieResult {
  id: number;
  domain: string;
  name: string;
  value: string;
  user_id: number;
}

const searchDomain = ref("");
const searchName = ref("");
const results = ref<CookieResult[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);

const handleSearch = async () => {
  loading.value = true;
  error.value = null;
  results.value = [];

  // TODO: Implement the actual API call to GET /api/v1/admin/cookies/search
  // For now, we'll use a timeout to simulate a network request
  console.log(`Searching for domain: ${searchDomain.value}, name: ${searchName.value}`);

  setTimeout(() => {
    // Mock response
    if (searchDomain.value === "example.com") {
      results.value = [
        {
          id: 1,
          domain: "example.com",
          name: "session",
          value: "abc-123-xyz",
          user_id: 1,
        },
        {
          id: 2,
          domain: "example.com",
          name: "user_pref",
          value: "theme=dark",
          user_id: 2,
        },
      ];
    }
    loading.value = false;
  }, 1000);
};
</script>
