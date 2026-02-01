

1. if a component file is more than 300 lines,  should refactor it into smaller component files into dedicated dir

2. there is no need to show success message or toast by calling message.success(...) or similar unless necessarily, only notify critical error

3. all date time related formatting, parsing should be put into lifelog-app-shared/src/lib/datetime.ts