import com.google.gson.JsonObject;

public class Function{
  public static JsonObject main(JsonObject args){
    System.out.println(args.toString());
    return args;
  }
}
