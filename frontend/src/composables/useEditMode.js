import { ref } from 'vue'

const editMode = ref(false)

export function useEditMode() {
  return { editMode }
}
