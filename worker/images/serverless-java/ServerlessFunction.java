import com.google.gson.JsonObject;
import com.google.gson.Gson;

import java.lang.reflect.Method;
import java.util.Scanner;

class ServerlessFunction {
    public static void main(String[] args) {
        SendData("started", "");

        Scanner input = new Scanner(System.in);
        Class<?> functionClass = getFunction(input);
        JsonObject success = new JsonObject();
        success.addProperty("type", "function_loaded");
        success.addProperty("data", true);
        System.out.println(success.toString());

        while (true) {
            String line = input.nextLine();
            try {
                JsonObject request = new Gson().fromJson(line, JsonObject.class);
                if (!request.get("type").getAsString().equals("request")) {
                    continue;
                }
                System.out.println("Handling: " + request.get("data").toString());
                JsonObject argument = request.getAsJsonObject("data");
                Object functionInstance = functionClass.getConstructor().newInstance();
                Method functionMethod = functionClass.getMethod("main", JsonObject.class);
                JsonObject result = (JsonObject)functionMethod.invoke(functionInstance, argument);
                JsonObject response = new JsonObject();
                response.addProperty("type", "response");
                response.add("data", result);
                System.out.println(response.toString());
            } catch(Exception e) {
                System.out.println(e);
                continue;
            }
        }

    }

    private static Class<?> getFunction(Scanner input) {
        while (true){
            String line = input.nextLine();
            try {
                JsonObject obj = new Gson().fromJson(line, JsonObject.class);

                String type = obj.get("type").getAsString();
                if (!type.equals("function")) {
                    continue;
                }
                String function = obj.get("data").getAsString();
                StringJarLoader loader = new StringJarLoader(function);
                return loader.findClass("Function");
            } catch(Exception e) {
                System.out.println("failed to load function");
                System.out.println(e);
                continue;
            }
        }
    }

    public static void SendData(String type, String data) {
        JsonObject out = new JsonObject();
        out.addProperty("type", type);
        out.addProperty("data", data);
        System.out.println(out.toString());
    }
}
